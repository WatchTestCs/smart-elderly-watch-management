package com.example.careold.controller;

import com.example.careold.domain.DeviceRequests;
import com.example.careold.service.DeviceService;
import com.fasterxml.jackson.databind.ObjectMapper;
import jakarta.validation.Valid;
import java.util.LinkedHashMap;
import org.springframework.web.bind.annotation.*;

@RestController
@RequestMapping("/restful/device")
public class DeviceController {
    private final DeviceService devices;
    private final ObjectMapper json;

    public DeviceController(DeviceService devices, ObjectMapper json) {
        this.devices = devices;
        this.json = json;
    }

    @PostMapping("/login")
    public LinkedHashMap login(@Valid @RequestBody DeviceRequests.Credentials request) {
        return devices.login(json.convertValue(request, LinkedHashMap.class));
    }

    @PostMapping("/bind")
    public LinkedHashMap bind(@Valid @RequestBody DeviceRequests.Binding request) {
        return devices.bind(json.convertValue(request, LinkedHashMap.class));
    }

    @PostMapping("/heartbeat")
    public LinkedHashMap heartbeat(@Valid @RequestBody DeviceRequests.Telemetry request) {
        return devices.heartbeat(json.convertValue(request, LinkedHashMap.class));
    }

    @PostMapping("/report")
    public LinkedHashMap report(@Valid @RequestBody DeviceRequests.Telemetry request) {
        return devices.report(json.convertValue(request, LinkedHashMap.class));
    }

    @PostMapping("/sos")
    public LinkedHashMap sos(@Valid @RequestBody DeviceRequests.Telemetry request) {
        return devices.sos(json.convertValue(request, LinkedHashMap.class));
    }

    @PostMapping("/list")
    public LinkedHashMap list(@Valid @RequestBody(required=false) DeviceRequests.ListQuery request) {
        return devices.list(json.convertValue(request, LinkedHashMap.class));
    }

    @PostMapping("/createBindCode")
    public LinkedHashMap createBindCode(@Valid @RequestBody DeviceRequests.CreateBinding request) {
        return devices.createBindCode(json.convertValue(request, LinkedHashMap.class));
    }

    @PostMapping("/bindCodeList")
    public LinkedHashMap bindCodeList(@Valid @RequestBody DeviceRequests.Elder request) {
        return devices.bindCodeList(json.convertValue(request, LinkedHashMap.class));
    }

    @PostMapping("/deleteBindCode")
    public LinkedHashMap deleteBindCode(@Valid @RequestBody DeviceRequests.BindingId request) {
        return devices.deleteBindCode(json.convertValue(request, LinkedHashMap.class));
    }

    @PostMapping("/create")
    public LinkedHashMap create(@Valid @RequestBody DeviceRequests.Create request) {
        return devices.create(json.convertValue(request, LinkedHashMap.class));
    }

    @PostMapping("/resetSecret")
    public LinkedHashMap resetSecret(@Valid @RequestBody DeviceRequests.DeviceId request) {
        return devices.resetSecret(json.convertValue(request, LinkedHashMap.class));
    }

    @PostMapping("/unbind")
    public LinkedHashMap unbind(@Valid @RequestBody DeviceRequests.DeviceId request) {
        return devices.unbind(json.convertValue(request, LinkedHashMap.class));
    }

    @PostMapping("/delete")
    public LinkedHashMap delete(@Valid @RequestBody DeviceRequests.DeviceId request) {
        return devices.delete(json.convertValue(request, LinkedHashMap.class));
    }

    @PostMapping("/handleAlarm")
    public LinkedHashMap handleAlarm(@Valid @RequestBody DeviceRequests.Elder request) {
        return devices.handleAlarm(json.convertValue(request, LinkedHashMap.class));
    }

    @PostMapping("/alertCounts")
    public LinkedHashMap alertCounts() { return devices.alertCounts(null); }

    @PostMapping("/alertDetail")
    public LinkedHashMap alertDetail(@Valid @RequestBody DeviceRequests.Alert request) {
        return devices.alertDetail(json.convertValue(request, LinkedHashMap.class));
    }

    @PostMapping("/resolveAlert")
    public LinkedHashMap resolveAlert(@Valid @RequestBody DeviceRequests.Alert request) {
        return devices.resolveAlert(json.convertValue(request, LinkedHashMap.class));
    }
}
