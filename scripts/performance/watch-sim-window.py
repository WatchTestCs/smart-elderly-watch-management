"""Check the load measurement window with its main thread at lower priority on Windows."""
import ctypes
from ctypes import wintypes
import importlib.util
import json
from pathlib import Path
import subprocess
import sys


def main():
    script = Path(__file__).with_name("watch-sim-database.py")
    spec = importlib.util.spec_from_file_location("database_fixture", script)
    fixture = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(fixture)
    args = fixture.options()
    if sys.platform != "win32" or not args.trace:
        raise SystemExit("This scheduling check requires Windows and --trace")
    fixture.options = lambda: args
    original_run = subprocess.run
    kernel = ctypes.WinDLL("kernel32", use_last_error=True)
    kernel.OpenThread.argtypes = [wintypes.DWORD, wintypes.BOOL, wintypes.DWORD]
    kernel.OpenThread.restype = wintypes.HANDLE
    kernel.ResumeThread.argtypes = [wintypes.HANDLE]
    kernel.ResumeThread.restype = wintypes.DWORD
    kernel.CloseHandle.argtypes = [wintypes.HANDLE]
    kernel.SetProcessAffinityMask.argtypes = [wintypes.HANDLE, ctypes.c_size_t]
    scheduling = []

    def controlled_run(command, **kwargs):
        if Path(command[0]).resolve() != args.load:
            return original_run(command, **kwargs)
        kwargs.pop("capture_output", None)
        timeout = kwargs.pop("timeout", None)
        # Set the initial thread before the workload can create its normal-priority workers.
        with subprocess.Popen(command, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                              creationflags=4, **kwargs) as process:
            try:
                code = (f"$taskMainThread=(Get-Process -Id {process.pid}).Threads[0]; "
                        "$taskMainThread.PriorityLevel=[System.Diagnostics.ThreadPriorityLevel]::Lowest; "
                        "@{id=$taskMainThread.Id;priority=$taskMainThread.PriorityLevel.ToString()} "
                        "| ConvertTo-Json -Compress")
                result = original_run(["C:/Program Files/PowerShell/7/pwsh.exe", "-NoProfile", "-Command", code],
                                      capture_output=True, text=True, check=True)
                thread = json.loads(result.stdout)
                if thread["priority"] != "Lowest":
                    raise RuntimeError("main thread priority was not changed")
                if not kernel.SetProcessAffinityMask(wintypes.HANDLE(int(process._handle)), 4):
                    raise ctypes.WinError(ctypes.get_last_error())
                handle = kernel.OpenThread(2, False, thread["id"])
                if not handle:
                    raise ctypes.WinError(ctypes.get_last_error())
                try:
                    if kernel.ResumeThread(handle) == 0xffffffff:
                        raise ctypes.WinError(ctypes.get_last_error())
                finally:
                    kernel.CloseHandle(handle)
                scheduling.append({"process_id": process.pid, "main_thread_priority": thread["priority"],
                                   "affinity_mask": 4})
                (args.output / "window-scheduling.json").write_text(
                    json.dumps(scheduling, indent=2) + "\n", encoding="utf-8", newline="\n")
                stdout, stderr = process.communicate(timeout=timeout)
            finally:
                if process.poll() is None:
                    process.kill()
                    process.wait()
        return subprocess.CompletedProcess(command, process.returncode, stdout, stderr)

    subprocess.run = controlled_run
    try:
        fixture.main()
    finally:
        subprocess.run = original_run


if __name__ == "__main__":
    main()
