package com.example.careold.domain;

import jakarta.validation.constraints.*;

public final class DeviceRequests {
    private DeviceRequests() { }

    public record Credentials(@NotBlank @Size(max=32) String deviceNo, @NotBlank @Size(max=128) String secret) { }
    public record Binding(@NotBlank @Size(max=32) String deviceNo, @NotBlank @Size(max=128) String secret,
                          @NotBlank @Size(max=20) String code) { }
    // Numeric strings retain the device protocol's support for JSON numbers and strings.
    // Strict integer, coordinate and timestamp validation belongs to the shared service.
    public record Telemetry(@NotBlank @Size(max=32) String deviceNo, @NotBlank @Size(max=128) String secret,
                            @Size(max=12) String hr, @Size(max=12) String bp, @Size(max=12) String step,
                            @Size(max=12) String sleep, @Size(max=1) String sleepState, @Size(max=20) String lng,
                            @Size(max=20) String lat, @Size(max=20) String source, @Size(max=19) String beTime,
                            @Size(max=1) String alarm, @Size(max=3) String battery, @Size(max=20) String appVersion) { }
    public record ListQuery(@Min(1) @Max(100000) Integer page, @Min(1) @Max(100) Integer pageSize,
                            @Size(max=100) String query, @Pattern(regexp="all|online|offline") String state) { }
    public record CreateBinding(@NotBlank @Pattern(regexp="[0-9]{1,9}") String oldId, @Size(max=3) String expireDays) { }
    public record Elder(@NotBlank @Pattern(regexp="[0-9]{1,9}") String oldId) { }
    public record BindingId(@NotBlank @Pattern(regexp="[0-9]{1,9}") String id) { }
    public record Create(@Size(max=32) String alias, @Size(max=32) String model, @Pattern(regexp="[0-9]{0,9}") String oldId) { }
    public record DeviceId(@NotBlank @Pattern(regexp="[0-9]{1,9}") String deviceId) { }
    public record Alert(@NotBlank @Pattern(regexp="[0-9]{1,9}") String deviceId,
                        @NotBlank @Pattern(regexp="person|rate|pre") String kind, @Pattern(regexp="[0-9]{0,9}") String recordId) { }
}
