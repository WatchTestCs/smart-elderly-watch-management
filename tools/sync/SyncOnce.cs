// 立即同步一次：把本地改动提交并推送到 GitHub
// 编译命令（Windows 自带 .NET Framework 编译器，无需安装任何软件）：
//   C:\Windows\Microsoft.NET\Framework64\v4.0.30319\csc.exe /nologo /optimize+ /codepage:65001 /out:立即同步到GitHub.exe tools\sync\SyncOnce.cs
// 用法：双击运行，或 命令行执行  立即同步到GitHub.exe --yes（跳过确认）

using System;
using System.Diagnostics;
using System.IO;
using System.Text;

internal static class SyncOnce
{
    private static string repo;      // 仓库根目录
    private static string gitExe;    // git.exe 全路径
    private static string logDir;    // 工具日志目录
    private static string logFile;

    private static int Main(string[] args)
    {
        Console.OutputEncoding = Encoding.UTF8;
        bool auto = HasFlag(args, "--yes") || HasFlag(args, "-y");
        Console.Title = "立即同步到 GitHub";

        // 以 exe 所在目录为准，这样双击、快捷方式、从别处调用都能找到仓库
        try { Directory.SetCurrentDirectory(AppDomain.CurrentDomain.BaseDirectory); }
        catch { }

        repo = FindRepoRoot();
        if (repo == null)
        {
            Fail("当前目录不在 Git 仓库里（找不到 .git 目录）。请把本程序放在仓库根目录运行。");
            return 2;
        }

        gitExe = FindGit();
        if (gitExe == null)
        {
            Fail("找不到 git.exe。请确认已安装 Git for Windows，或用它的安装路径加入 PATH 后重试。");
            return 2;
        }

        logDir = Path.Combine(repo, "工具日志");
        logFile = Path.Combine(logDir, "同步日志.txt");

        Line('=', 60);
        Console.WriteLine("立即同步到 GitHub");
        Line('=', 60);
        Console.WriteLine("仓库: " + repo);
        Console.WriteLine("远端: " + (Git("remote get-url origin").Out ?? "（未设置）").Trim());
        Console.WriteLine();

        // ---------- 1. 检查是否有东西要传 ----------
        // -c core.quotepath=false 让中文文件名正常显示（只影响本次调用，不改仓库配置）
        GitResult status = Git("-c core.quotepath=false status --porcelain");
        if (status.Out == null)
        {
            Fail("读取仓库状态失败：\n" + status.Err);
            return 2;
        }

        int pending = 0;
        foreach (string l in status.Out.Split('\n'))
            if (l.Trim().Length > 0) pending++;

        int aheadCount;
        bool needPush = IsAheadOfUpstream(out aheadCount);

        if (pending == 0 && !needPush)
        {
            Console.WriteLine("没有任何改动，远端也已是最新，无需同步。");
            Console.WriteLine();
            Pause(auto);
            return 0;
        }

        // ---------- 2. 展示将要上传的文件（已按 .gitignore 过滤） ----------
        if (pending > 0)
        {
            Console.WriteLine("将要上传的改动（共 " + pending + " 项，已按 .gitignore 过滤）：");
            Console.WriteLine("------------------------------------------------------------");
            string[] lines = status.Out.Replace("\r\n", "\n").Split('\n');
            int shown = 0;
            foreach (string l in lines)
            {
                if (l.Trim().Length == 0) continue;
                if (shown++ < 40) Console.WriteLine("  " + l);
            }
            if (pending > 40) Console.WriteLine("  … 其余 " + (pending - 40) + " 项省略");
            Console.WriteLine("------------------------------------------------------------");
            Console.WriteLine("提醒：确认里面没有密码、密钥、数据库导出、几百 MB 的大文件。");
            Console.WriteLine();
        }
        else
        {
            Console.WriteLine("工作树干净，但本地有 " + aheadCount + " 个提交还没推到远端，将只执行推送。");
            Console.WriteLine();
        }

        if (!auto)
        {
            Console.Write("确认上传？输入 y 回车继续，其它键取消：");
            string ans = (Console.ReadLine() ?? "").Trim();
            if (!ans.Equals("y", StringComparison.OrdinalIgnoreCase)
                && !ans.Equals("yes", StringComparison.OrdinalIgnoreCase))
            {
                Console.WriteLine("已取消，什么都没有改变。");
                Pause(auto);
                return 0;
            }
        }

        // ---------- 3. 先探测远端，避免后面合并不上 ----------
        Console.WriteLine();
        Console.WriteLine("[1/3] 获取远端最新状态 …");
        GitResult fetch = Git("fetch origin");
        if (fetch.Code != 0)
        {
            Console.WriteLine("  拉取失败，可能是网络不通。为安全起见已停止，没有提交任何东西。");
            Console.WriteLine("  详情：" + OneLine(fetch.Err));
            Log("fetch 失败: " + OneLine(fetch.Err));
            Pause(auto);
            return 3;
        }

        // 注意：这里用 origin/main 的跟踪引用判断，而不是 fetch 的文本输出
        // （git fetch 不带 refspec 时只写 FETCH_HEAD，不会更新 origin/main）。
        string behindText = (Git("rev-list --count HEAD..origin/main").Out ?? "0").Trim();
        int behind;
        if (!int.TryParse(behindText, out behind)) behind = 0;
        if (behind > 0)
        {
            Console.WriteLine("  远端有 " + behind + " 个本地没有的提交（有人在网页上改过，或别处推送过）。");
            Console.WriteLine("  为安全起见已停止，没有提交任何东西。请告诉我，我会先合并再推，");
            Console.WriteLine("  绝不会用强推去覆盖别人的东西。");
            Log("远端领先 " + behind + " 个提交，已停止");
            Pause(auto);
            return 4;
        }

        // ---------- 4. 提交 ----------
        if (pending > 0)
        {
            Console.WriteLine("[2/3] 提交改动 …");
            GitResult add = Git("add -A");
            if (add.Code != 0)
            {
                Console.WriteLine("  暂存失败，已停止：" + OneLine(add.Err));
                Log("add 失败: " + OneLine(add.Err));
                Pause(auto);
                return 5;
            }
            string msg = "同步 " + DateTime.Now.ToString("yyyy-MM-dd HH:mm:ss") + "（" + pending + " 项改动）";
            GitResult commit = Git("commit -m \"" + msg + "\"");
            if (commit.Code != 0)
            {
                Console.WriteLine("  提交失败，已停止：" + OneLine(commit.Err));
                Log("commit 失败: " + OneLine(commit.Err));
                Pause(auto);
                return 5;
            }
            Console.WriteLine("  已提交：" + msg);
        }
        else
        {
            Console.WriteLine("[2/3] 无需提交（没有文件改动）。");
        }

        // ---------- 5. 推送 ----------
        Console.WriteLine("[3/3] 推送到 origin/main …");
        GitResult push = Git("push origin main");
        Console.WriteLine("  " + OneLine(push.Out));
        if (push.Code != 0)
        {
            Console.WriteLine();
            Console.WriteLine("  推送失败。最常见的两种原因：");
            Console.WriteLine("   1) 网络/DNS 不通 —— 稍后再点一次即可；");
            Console.WriteLine("   2) 登录凭据过期 —— 在命令行执行一次 git push，按提示登录后就会记住。");
            Console.WriteLine("  详情：" + OneLine(push.Err));
            Log("push 失败: " + OneLine(push.Err));
            Console.WriteLine();
            Console.WriteLine("  注意：提交已经做在本地了，不会丢；网络恢复后再点一次本程序就会补推上去。");
            Pause(auto);
            return 6;
        }

        string head = (Git("log -1 --oneline").Out ?? "").Trim();
        Console.WriteLine();
        Line('=', 60);
        Console.WriteLine("同步完成 ✔  远端 main 现在是：" + head);
        Console.WriteLine("查看：https://github.com/WatchTestCs/smart-elderly-watch-management");
        Line('=', 60);
        Log("同步成功: " + head);
        Pause(auto);
        return 0;
    }

    // ================= 辅助 =================

    private static bool HasFlag(string[] args, string flag)
    {
        foreach (string a in args)
            if (string.Equals(a, flag, StringComparison.OrdinalIgnoreCase)) return true;
        return false;
    }

    /// <summary>从当前目录（或其子目录）向上找含 .git 的目录。</summary>
    private static string FindRepoRoot()
    {
        string dir = Directory.GetCurrentDirectory();
        for (int i = 0; i < 16 && !string.IsNullOrEmpty(dir); i++)
        {
            if (Directory.Exists(Path.Combine(dir, ".git"))) return dir;
            DirectoryInfo parent = Directory.GetParent(dir);
            if (parent == null) break;
            dir = parent.FullName;
        }
        return null;
    }

    private static string FindGit()
    {
        string[] candidates =
        {
            @"C:\Program Files\Git\cmd\git.exe",
            @"C:\Program Files (x86)\Git\cmd\git.exe",
            Environment.GetEnvironmentVariable("ProgramFiles") == null
                ? null : Path.Combine(Environment.GetEnvironmentVariable("ProgramFiles"), @"Git\cmd\git.exe"),
            Environment.GetEnvironmentVariable("ProgramW6432") == null
                ? null : Path.Combine(Environment.GetEnvironmentVariable("ProgramW6432"), @"Git\cmd\git.exe"),
            Environment.GetEnvironmentVariable("LOCALAPPDATA") == null
                ? null : Path.Combine(Environment.GetEnvironmentVariable("LOCALAPPDATA"), @"Programs\Git\cmd\git.exe")
        };
        foreach (string c in candidates)
            if (!string.IsNullOrEmpty(c) && File.Exists(c)) return c;
        return "git"; // 交给 PATH 解析
    }

    private struct GitResult { public int Code; public string Out; public string Err; }

    private static GitResult Git(string gitArgs)
    {
        GitResult r = new GitResult();
        try
        {
            ProcessStartInfo psi = new ProcessStartInfo("cmd.exe",
                "/c \"\"" + gitExe + "\" " + gitArgs + " 2>&1\"");
            psi.WorkingDirectory = repo;
            psi.UseShellExecute = false;
            psi.CreateNoWindow = true;
            psi.StandardOutputEncoding = Encoding.UTF8;
            psi.StandardErrorEncoding = Encoding.UTF8;
            psi.RedirectStandardOutput = true;
            psi.RedirectStandardError = true;
            psi.EnvironmentVariables["GIT_PAGER"] = "cat";
            psi.EnvironmentVariables["GIT_TERMINAL_PROMPT"] = "0";

            using (Process p = Process.Start(psi))
            {
                r.Out = p.StandardOutput.ReadToEnd();
                r.Err = p.StandardError.ReadToEnd();
                p.WaitForExit();
                r.Code = p.ExitCode;
            }
        }
        catch (Exception ex)
        {
            r.Code = -1;
            r.Out = "";
            r.Err = ex.Message;
        }
        return r;
    }

    private static bool IsAheadOfUpstream(out int ahead)
    {
        ahead = 0;
        GitResult r = Git("rev-list --count origin/main..HEAD");
        if (r.Code != 0 || r.Out == null) return false;
        int.TryParse(r.Out.Trim(), out ahead);
        return ahead > 0;
    }

    private static string OneLine(string s)
    {
        if (string.IsNullOrEmpty(s)) return "";
        return s.Replace("\r", " ").Replace("\n", " ").Trim();
    }

    private static void Line(char c, int n) { Console.WriteLine(new string(c, n)); }

    private static void Log(string text)
    {
        try
        {
            if (!Directory.Exists(logDir)) Directory.CreateDirectory(logDir);
            File.AppendAllText(logFile, DateTime.Now.ToString("yyyy-MM-dd HH:mm:ss") + "  " + text + Environment.NewLine, Encoding.UTF8);
        }
        catch { /* 日志写不了不影响主流程 */ }
    }

    private static void Fail(string msg)
    {
        Line('=', 60);
        Console.WriteLine("无法同步：" + msg);
        Line('=', 60);
        Pause(false);
    }

    private static void Pause(bool auto)
    {
        if (!auto && Environment.UserInteractive)
        {
            Console.WriteLine();
            Console.Write("按回车键关闭窗口 …");
            Console.ReadLine();
        }
    }
}
