// Generated from the published Go OpenAPI by npm run api:generate. Do not edit.
export interface paths {
    "/api/v1/activity/alerts/device-counts": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * activity/alerts/device-counts
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_vitals.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: components["requestBodies"]["JsonObject"];
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            /** @constant */
                            returnCode: "1111";
                            sleep: number;
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/activity/chart": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * activity/chart
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_vitals.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: components["requestBodies"]["ResidentId"];
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            activityRecords?: unknown;
                            createTime?: unknown;
                            /** @constant */
                            returnCode: "1111";
                            sleep?: unknown;
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/activity/history": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * activity/history
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_vitals.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: components["requestBodies"]["ResidentId"];
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            activityRecords?: unknown;
                            /** @constant */
                            returnCode: "1111";
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/activity/latest": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * activity/latest
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_vitals.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        name?: unknown;
                        page?: unknown;
                        pageSize?: unknown;
                        timeFirst?: unknown;
                        timeLast?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            activityRecords?: unknown;
                            page?: unknown;
                            pageSize?: unknown;
                            /** @constant */
                            returnCode: "1111";
                            total?: unknown;
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/alert-reviews": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * alert-reviews
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_capability_routes.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        expectedReviewId?: number;
                        kind?: string;
                        reason?: string;
                        recordId?: number;
                        verdict?: string;
                    };
                };
            };
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            alarmStateChanged?: unknown;
                            /** @constant */
                            returnCode: "1111";
                            reviewId?: unknown;
                            verdict?: unknown;
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/alert-reviews/search": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * alert-reviews/search
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_capability_routes.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        after?: number;
                        entity?: string;
                        kind?: string;
                        recordId?: number;
                    };
                };
            };
            responses: {
                200: components["responses"]["ItemsSuccess"];
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/auth/login": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * auth/login
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_auth.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        otp?: string;
                        password?: string;
                        phone?: string;
                        recoveryCode?: string;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            authenticated?: unknown;
                            /** @constant */
                            returnCode: "1111";
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/auth/logout": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * auth/logout
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_auth.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: components["requestBodies"]["JsonObject"];
            responses: {
                200: components["responses"]["Success"];
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/auth/session": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * auth/session
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_auth.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        get: {
            parameters: {
                query?: never;
                header?: never;
                path?: never;
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            account?: unknown;
                            authenticated?: unknown;
                            csrfHeader?: unknown;
                            csrfToken?: unknown;
                            /** @constant */
                            returnCode: "1111";
                            setupRequired?: unknown;
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/care/accessAudit": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * care/accessAudit
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_care_incident.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: components["requestBodies"]["JsonObject"];
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            accesses?: unknown;
                            /** @constant */
                            returnCode: "1111";
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/care/assignDuty": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * care/assignDuty
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_care_incident.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        adminId?: unknown;
                        duty?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            adminId?: unknown;
                            duty?: unknown;
                            /** @constant */
                            returnCode: "1111";
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/care/diagnostics": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * care/diagnostics
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_care_incident.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: components["requestBodies"]["JsonObject"];
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            analytics?: unknown;
                            failedNotifications?: unknown;
                            note?: unknown;
                            notificationConfigured?: unknown;
                            oldestOutstandingAt?: unknown;
                            overdueIncidents?: unknown;
                            pendingNotifications?: unknown;
                            primaryEngine?: unknown;
                            receiptConfigured?: unknown;
                            /** @constant */
                            returnCode: "1111";
                            staleSending?: unknown;
                            workerHealthy?: unknown;
                            workerLastSeenAt?: unknown;
                            workerStartedAt?: unknown;
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/care/events": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * care/events
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_care_incident.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        id?: unknown;
                        page?: unknown;
                        pageSize?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            events?: unknown;
                            page?: unknown;
                            pageSize?: unknown;
                            /** @constant */
                            returnCode: "1111";
                            total?: unknown;
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/care/incidents": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * care/incidents
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_care_incident.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        page?: unknown;
                        pageSize?: unknown;
                        state?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            actorId?: unknown;
                            duty?: unknown;
                            items?: unknown;
                            page?: unknown;
                            pageSize?: unknown;
                            /** @constant */
                            returnCode: "1111";
                            total?: unknown;
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/care/receipt": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * care/receipt
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_care_incident.go. Device body authentication or signed gateway receipt; not anonymous access.
         */
        post: {
            parameters: {
                query?: never;
                header?: never;
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        notificationId?: unknown;
                        status?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            notificationId?: unknown;
                            /** @constant */
                            returnCode: "1111";
                            status?: unknown;
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/care/staff": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * care/staff
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_care_incident.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: components["requestBodies"]["JsonObject"];
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            /** @constant */
                            returnCode: "1111";
                            staff?: unknown;
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/care/transition": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * care/transition
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_care_incident.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        action?: unknown;
                        id?: unknown;
                        note?: unknown;
                        verdict?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            id?: unknown;
                            /** @constant */
                            returnCode: "1111";
                            state?: unknown;
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/desktop/exit": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * desktop/exit
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_auth.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header?: never;
                path?: never;
                cookie?: never;
            };
            requestBody: components["requestBodies"]["JsonObject"];
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            closing?: unknown;
                            /** @constant */
                            returnCode: "1111";
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/device-command-profiles": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * device-command-profiles
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_capability_routes.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        deviceId?: number;
                        enabled?: boolean;
                        protocol?: string;
                        wireType?: string;
                    };
                };
            };
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            deviceId?: unknown;
                            enabled?: unknown;
                            /** @constant */
                            returnCode: "1111";
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/device-commands": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * device-commands
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_capability_routes.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        action?: string;
                        deviceId?: number;
                        requestKey?: string;
                        ttlSeconds?: number;
                    };
                };
            };
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            commandId?: unknown;
                            duplicate?: unknown;
                            /** @constant */
                            returnCode: "1111";
                            status?: unknown;
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/device-commands/search": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * device-commands/search
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_capability_routes.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        after?: number;
                        deviceId?: number;
                    };
                };
            };
            responses: {
                200: components["responses"]["ItemsSuccess"];
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/device/alertCounts": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * device/alertCounts
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_resident_devices.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: components["requestBodies"]["JsonObject"];
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            items?: unknown;
                            personTotal?: unknown;
                            preTotal?: unknown;
                            rateTotal?: unknown;
                            /** @constant */
                            returnCode: "1111";
                            total?: unknown;
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/device/alertDetail": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * device/alertDetail
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_resident_devices.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        deviceId?: unknown;
                        kind?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            advice?: unknown;
                            alias?: unknown;
                            detail?: unknown;
                            deviceId?: unknown;
                            deviceNo?: unknown;
                            hasAlert?: unknown;
                            kind?: unknown;
                            kindText?: unknown;
                            message?: unknown;
                            occurTime?: unknown;
                            oldPersonName?: unknown;
                            personHandled?: unknown;
                            preHandled?: unknown;
                            rateHandled?: unknown;
                            reason?: unknown;
                            recordId?: unknown;
                            /** @constant */
                            returnCode: "1111";
                            threshold?: unknown;
                            value?: unknown;
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/device/bind": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * device/bind
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_resident_devices.go. Device body authentication or signed gateway receipt; not anonymous access.
         */
        post: {
            parameters: {
                query?: never;
                header?: never;
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        code?: unknown;
                        deviceNo?: unknown;
                        secret?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            oldId?: unknown;
                            oldPersonName?: unknown;
                            /** @constant */
                            returnCode: "1111";
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/device/bindCodeList": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * device/bindCodeList
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_resident_devices.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: components["requestBodies"]["ResidentId"];
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            bindCodes?: unknown;
                            /** @constant */
                            returnCode: "1111";
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/device/commandAck": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * device/commandAck
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_capability_routes.go. Device body authentication or signed gateway receipt; not anonymous access.
         */
        post: {
            parameters: {
                query?: never;
                header?: never;
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        bindingVersion?: unknown;
                        commandId?: unknown;
                        deliveryToken?: unknown;
                        deviceNo?: unknown;
                        result?: unknown;
                        secret?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            commandId?: unknown;
                            /** @constant */
                            returnCode: "1111";
                            status?: unknown;
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/device/commandPull": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * device/commandPull
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_capability_routes.go. Device body authentication or signed gateway receipt; not anonymous access.
         */
        post: {
            parameters: {
                query?: never;
                header?: never;
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        bindingVersion?: unknown;
                        deviceNo?: unknown;
                        secret?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            command?: unknown;
                            /** @constant */
                            returnCode: "1111";
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/device/create": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * device/create
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_resident_devices.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        alias?: unknown;
                        model?: unknown;
                        oldId?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            deviceId?: unknown;
                            deviceNo?: unknown;
                            /** @constant */
                            returnCode: "1111";
                            secret?: unknown;
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/device/createBindCode": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * device/createBindCode
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_resident_devices.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        createUser?: unknown;
                        expireDays?: unknown;
                        oldId?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            code?: unknown;
                            expireTime?: unknown;
                            oldId?: unknown;
                            oldPersonName?: unknown;
                            /** @constant */
                            returnCode: "1111";
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/device/delete": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * device/delete
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_resident_devices.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: components["requestBodies"]["DeviceId"];
            responses: {
                200: components["responses"]["Success"];
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/device/deleteBindCode": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * device/deleteBindCode
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_resident_devices.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        id?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                200: components["responses"]["Success"];
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/device/fencePoint": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * device/fencePoint
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_capability_routes.go. Device body authentication or signed gateway receipt; not anonymous access.
         */
        post: {
            parameters: {
                query?: never;
                header?: never;
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        accuracy?: unknown;
                        bindingVersion?: unknown;
                        coordinateSystem?: unknown;
                        deviceNo?: unknown;
                        occurredAt?: unknown;
                        point?: unknown;
                        secret?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            events?: unknown;
                            /** @constant */
                            returnCode: "1111";
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/device/healthBatch": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * device/healthBatch
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_health.go. Device body authentication or signed gateway receipt; not anonymous access.
         */
        post: {
            parameters: {
                query?: never;
                header?: never;
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        bindingVersion?: unknown;
                        deviceNo?: unknown;
                        samples?: unknown;
                        secret?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            accepted?: unknown;
                            duplicates?: unknown;
                            /** @constant */
                            returnCode: "1111";
                            writeRevision?: unknown;
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/device/heartbeat": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * device/heartbeat
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_resident_devices.go. Device body authentication or signed gateway receipt; not anonymous access.
         */
        post: {
            parameters: {
                query?: never;
                header?: never;
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        appVersion?: unknown;
                        battery?: unknown;
                        deviceNo?: unknown;
                        secret?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            bindingVersion?: unknown;
                            bound?: unknown;
                            oldId?: unknown;
                            pendingAlarm?: unknown;
                            reportInterval?: unknown;
                            /** @constant */
                            returnCode: "1111";
                            serverTime?: unknown;
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/device/list": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * device/list
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_resident_devices.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        page?: unknown;
                        pageSize?: unknown;
                        query?: unknown;
                        state?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            devices?: unknown;
                            onlineCount?: unknown;
                            page?: unknown;
                            pageSize?: unknown;
                            /** @constant */
                            returnCode: "1111";
                            total?: unknown;
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/device/login": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * device/login
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_resident_devices.go. Device body authentication or signed gateway receipt; not anonymous access. HarmonyOS clients send optional platform=harmonyos with protocolVersion=1; both are required together when platform is supplied. Authenticated success returns protocolVersion=1 and maxBatchSize=200 plus the existing bindingVersion/owner. Unsupported platform/version returns 400. Omit platform for the legacy login contract. Invalid credentials return 401 with code=DEVICE_AUTH_FAILED; disabled devices return 403.
         */
        post: {
            parameters: {
                query?: never;
                header?: never;
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        deviceNo: string;
                        platform?: string;
                        protocolVersion?: number;
                        secret: string;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            alias?: unknown;
                            bindingVersion?: unknown;
                            bound?: unknown;
                            deviceId?: unknown;
                            deviceNo?: unknown;
                            maxBatchSize?: unknown;
                            oldId?: unknown;
                            oldPersonName?: unknown;
                            protocolVersion?: unknown;
                            /** @constant */
                            returnCode: "1111";
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/device/report": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * device/report
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_resident_devices.go. Device body authentication or signed gateway receipt; not anonymous access.
         */
        post: {
            parameters: {
                query?: never;
                header?: never;
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        alarm?: unknown;
                        appVersion?: unknown;
                        battery?: unknown;
                        beTime?: unknown;
                        bp?: unknown;
                        deviceNo?: unknown;
                        hr?: unknown;
                        lat?: unknown;
                        lng?: unknown;
                        positionAccuracy?: number;
                        /** Format: date-time */
                        positionSampledAt?: string;
                        /** @enum {string} */
                        positionSource?: "gps" | "network" | "fused" | "manual" | "simulated" | "unknown";
                        requestId?: unknown;
                        secret?: unknown;
                        sleep?: unknown;
                        sleepState?: unknown;
                        source?: unknown;
                        step?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            message?: unknown;
                            preState?: unknown;
                            queued?: unknown;
                            rateState?: unknown;
                            /** @constant */
                            returnCode: "1111";
                            serverTime?: unknown;
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/device/resetSecret": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * device/resetSecret
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_resident_devices.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: components["requestBodies"]["DeviceId"];
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            /** @constant */
                            returnCode: "1111";
                            secret?: unknown;
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/device/resolveAlert": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * device/resolveAlert
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_resident_devices.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        deviceId?: unknown;
                        kind?: unknown;
                        recordId?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            handledReason?: unknown;
                            message?: unknown;
                            /** @constant */
                            returnCode: "1111";
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/device/sos": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * device/sos
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_resident_devices.go. Device body authentication or signed gateway receipt; not anonymous access.
         */
        post: {
            parameters: {
                query?: never;
                header?: never;
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        alarm?: unknown;
                        bindingVersion?: unknown;
                        deviceNo?: unknown;
                        lat?: unknown;
                        lng?: unknown;
                        positionAccuracy?: number;
                        /** Format: date-time */
                        positionSampledAt?: string;
                        /** @enum {string} */
                        positionSource?: "gps" | "network" | "fused" | "manual" | "simulated" | "unknown";
                        requestId?: unknown;
                        secret?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            duplicate?: unknown;
                            incidentId?: unknown;
                            incidentState?: unknown;
                            message?: unknown;
                            notificationState?: unknown;
                            /** @constant */
                            returnCode: "1111";
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/device/unbind": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * device/unbind
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_resident_devices.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: components["requestBodies"]["DeviceId"];
            responses: {
                200: components["responses"]["Success"];
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/family/add": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * family/add
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_family.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        familyAddress?: unknown;
                        familyName?: unknown;
                        familyPhone?: unknown;
                        familyRelation?: unknown;
                        familySex?: unknown;
                        oldPhone?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                200: components["responses"]["Success"];
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/family/deleteFamily": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * family/deleteFamily
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_family.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        familyId?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                200: components["responses"]["Success"];
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/family/select": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * family/select
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_family.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        familyName?: unknown;
                        page?: unknown;
                        pageSize?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            familyDtos?: unknown;
                            page?: unknown;
                            pageSize?: unknown;
                            /** @constant */
                            returnCode: "1111";
                            total?: unknown;
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/family/selectTwo": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * family/selectTwo
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_family.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        familyPhone?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            familyDto2?: unknown;
                            /** @constant */
                            returnCode: "1111";
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/family/updateFamily": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * family/updateFamily
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_family.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        familyAddress?: unknown;
                        familyId?: unknown;
                        familyName?: unknown;
                        familyPhone?: unknown;
                        familyRelation?: unknown;
                        familySex?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                200: components["responses"]["Success"];
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/family/updatePassword": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * family/updatePassword
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_family.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        familyPassword?: unknown;
                        familyPhone?: unknown;
                        oldPassword?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                200: components["responses"]["Success"];
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/family/updateTouPic": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * family/updateTouPic
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_family.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        familyPhone?: unknown;
                        familyTouPic?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                200: components["responses"]["Success"];
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/family/updateTwo": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * family/updateTwo
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_family.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        familyAddress?: unknown;
                        familyId?: unknown;
                        familyName?: unknown;
                        familyPhone?: unknown;
                        familySex?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                200: components["responses"]["Success"];
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/fence-events/search": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * fence-events/search
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_capability_routes.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        after?: unknown;
                        oldId?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                200: components["responses"]["ItemsSuccess"];
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/fences": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * fences
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_capability_routes.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        coordinateSystem?: unknown;
                        coordinates?: unknown;
                        deviceId?: unknown;
                        dwellSeconds?: unknown;
                        enabled?: unknown;
                        id?: unknown;
                        maxAccuracy?: unknown;
                        minPoints?: unknown;
                        name?: unknown;
                        revision?: unknown;
                        timeBegin?: unknown;
                        timeEnd?: unknown;
                        timezone?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            deviceApplied?: unknown;
                            id?: unknown;
                            /** @constant */
                            returnCode: "1111";
                            revision?: unknown;
                            state?: unknown;
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/fences/search": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * fences/search
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_capability_routes.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: components["requestBodies"]["DeviceId"];
            responses: {
                200: components["responses"]["ItemsSuccess"];
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/gallery/list": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * gallery/list
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_social.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        id?: unknown;
                        picName?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            galleryImages?: unknown;
                            /** @constant */
                            returnCode: "1111";
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/gallery/replace": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * gallery/replace
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_social.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        fileListName?: string[];
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                200: components["responses"]["Success"];
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/guardians/create": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * guardians/create
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_user.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        customerAddress?: unknown;
                        customerAge?: unknown;
                        customerName?: unknown;
                        customerPhone?: unknown;
                        customerSex?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                200: components["responses"]["Success"];
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/guardians/delete": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * guardians/delete
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_user.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        customerId?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                200: components["responses"]["Success"];
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/guardians/list": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * guardians/list
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_user.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        customerName?: unknown;
                        page?: unknown;
                        pageSize?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            customers?: unknown;
                            page?: unknown;
                            pageSize?: unknown;
                            /** @constant */
                            returnCode: "1111";
                            total?: unknown;
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/guardians/update": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * guardians/update
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_user.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        customerAddress?: unknown;
                        customerAge?: unknown;
                        customerId?: unknown;
                        customerName?: unknown;
                        customerPhone?: unknown;
                        customerSex?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                200: components["responses"]["Success"];
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/health/alerts/device-counts": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * health/alerts/device-counts
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_vitals.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: components["requestBodies"]["JsonObject"];
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            bloodPressure: number;
                            heartRate: number;
                            /** @constant */
                            returnCode: "1111";
                            sleep: number;
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/health/changes": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * health/changes
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_health.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        after?: unknown;
                        limit?: unknown;
                        oldId?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            changes?: unknown;
                            hasMore?: unknown;
                            nextCursor?: unknown;
                            oldId?: unknown;
                            /** @constant */
                            returnCode: "1111";
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/health/daily-query": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * health/daily-query
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_capability_routes.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        day?: string;
                        days?: number;
                        metric?: string;
                        offsetMinutes?: number;
                        oldId?: number;
                        semantics?: string;
                        source?: string;
                        sourceDevice?: string;
                    };
                };
            };
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            algorithmVersion?: unknown;
                            items?: unknown;
                            metric?: unknown;
                            offsetMinutes?: unknown;
                            /** @constant */
                            returnCode: "1111";
                            semantics?: unknown;
                            source?: unknown;
                            sourceDevice?: unknown;
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/health/latest": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * health/latest
         * @description Latest nondeleted stored sample per exact source/sourceDevice/metric/semantics. Order by exact sample.start descending, then revision descending. Intervals retain both endpoints; daily totals are snapshots, not today's aggregate. FAMILY may omit oldId and is checked against current ownership; ADMIN supplies oldId. Historical binding/version and sampling time are preserved; no freshness guarantee or cross-source merging.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        oldId?: number;
                    };
                };
            };
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            items: {
                                bindingVersion: number;
                                deviceId: number | null;
                                oldId: number;
                                revision: number;
                                sample: {
                                    deleted?: boolean;
                                    /** Format: date-time */
                                    end: string;
                                    groupId?: string;
                                    metadata?: {
                                        [key: string]: string;
                                    };
                                    metric: string;
                                    recordId: string;
                                    semantics: string;
                                    source: string;
                                    sourceDevice: string;
                                    sourceVersion?: string;
                                    stage?: string;
                                    /** Format: date-time */
                                    start: string;
                                    timeBasis?: string;
                                    unit: string;
                                    value: number | null;
                                    version: number;
                                };
                                sampleId: string;
                            }[];
                            oldId: number;
                            /** @constant */
                            returnCode: "1111";
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/health/statistics": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * health/statistics
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_health.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        from?: unknown;
                        metric?: unknown;
                        oldId?: unknown;
                        semantics?: unknown;
                        source?: unknown;
                        sourceDevice?: unknown;
                        to?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            metric?: unknown;
                            /** @constant */
                            returnCode: "1111";
                            semantics?: unknown;
                            source?: unknown;
                            sourceDevice?: unknown;
                            statistics?: unknown;
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/image/download": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * image/download
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_social.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        get: {
            parameters: {
                query: {
                    name: string;
                };
                header?: never;
                path?: never;
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                /** @description Image bytes */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "image/*": string;
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/image/saveImage": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * image/saveImage
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_social.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "multipart/form-data": {
                        /** Format: binary */
                        file: string;
                    };
                };
            };
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            filename?: unknown;
                            /** @constant */
                            returnCode: "1111";
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/inviteCode/checkCode": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * inviteCode/checkCode
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_family.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        code?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            expireTime?: unknown;
                            oldPersonName?: unknown;
                            remainDays?: unknown;
                            /** @constant */
                            returnCode: "1111";
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/inviteCode/create": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * inviteCode/create
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_family.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        count?: unknown;
                        createUser?: unknown;
                        expireDays?: unknown;
                        oldId?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            codes?: unknown;
                            expireTime?: unknown;
                            oldId?: unknown;
                            oldPersonName?: unknown;
                            /** @constant */
                            returnCode: "1111";
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/inviteCode/delete": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * inviteCode/delete
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_family.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        id?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                200: components["responses"]["Success"];
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/inviteCode/join": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * inviteCode/join
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_family.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        code?: unknown;
                        familyAddress?: unknown;
                        familyName?: unknown;
                        familyPassword?: unknown;
                        familyPhone?: unknown;
                        familyRelation?: unknown;
                        familySex?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            familyId?: unknown;
                            familyName?: unknown;
                            oldPersonName?: unknown;
                            /** @constant */
                            returnCode: "1111";
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/inviteCode/list": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * inviteCode/list
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_family.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        page?: unknown;
                        pageSize?: unknown;
                        state?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            inviteCodes?: unknown;
                            page?: unknown;
                            pageSize?: unknown;
                            /** @constant */
                            returnCode: "1111";
                            total?: unknown;
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/inviteCode/oldPersonList": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * inviteCode/oldPersonList
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_family.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: components["requestBodies"]["JsonObject"];
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            oldPeople?: unknown;
                            /** @constant */
                            returnCode: "1111";
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/locations/alerts/resolve-latest": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * locations/alerts/resolve-latest
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_resident_devices.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: components["requestBodies"]["ResidentId"];
            responses: {
                200: components["responses"]["Success"];
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/locations/alerts/summary": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * locations/alerts/summary
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_resident_devices.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: components["requestBodies"]["JsonObject"];
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            activeAlerts?: unknown;
                            /** @constant */
                            returnCode: "1111";
                            totalResidents?: unknown;
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/monitoring-profiles/create": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * monitoring-profiles/create
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_resident_devices.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        bpf?: unknown;
                        bpl?: unknown;
                        deviceId?: unknown;
                        hrf?: unknown;
                        hrl?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                200: components["responses"]["Success"];
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/monitoring-profiles/delete": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * monitoring-profiles/delete
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_resident_devices.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: components["requestBodies"]["DeviceId"];
            responses: {
                200: components["responses"]["Success"];
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/monitoring-profiles/list": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * monitoring-profiles/list
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_resident_devices.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        oldName?: unknown;
                        page?: unknown;
                        pageSize?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            page?: unknown;
                            pageSize?: unknown;
                            /** @constant */
                            returnCode: "1111";
                            total?: unknown;
                            watchDtos?: unknown;
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/monitoring-profiles/update": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * monitoring-profiles/update
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_resident_devices.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        bpf?: unknown;
                        bpl?: unknown;
                        deviceId?: unknown;
                        hrf?: unknown;
                        hrl?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                200: components["responses"]["Success"];
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/notice-comments/create": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * notice-comments/create
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_social.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        said?: unknown;
                        textId?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                200: components["responses"]["Success"];
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/notice-comments/delete": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * notice-comments/delete
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_social.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        saidId?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                200: components["responses"]["Success"];
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/notice-comments/list": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * notice-comments/list
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_social.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: components["requestBodies"]["NoticeId"];
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            comments?: unknown;
                            /** @constant */
                            returnCode: "1111";
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/notice-comments/reply": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * notice-comments/reply
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_social.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        back?: unknown;
                        saidId?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                200: components["responses"]["Success"];
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/notice-comments/search": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * notice-comments/search
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_social.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        page?: unknown;
                        pageSize?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            moderatedComments?: unknown;
                            page?: unknown;
                            pageSize?: unknown;
                            /** @constant */
                            returnCode: "1111";
                            total?: unknown;
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/notice-comments/toggle-like": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * notice-comments/toggle-like
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_social.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        saidId?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                200: components["responses"]["Success"];
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/notices/create": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * notices/create
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_social.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        fileListName?: string[];
                        textMain?: unknown;
                        title?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                200: components["responses"]["Success"];
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/notices/delete": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * notices/delete
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_social.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: components["requestBodies"]["NoticeId"];
            responses: {
                200: components["responses"]["Success"];
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/notices/detail": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * notices/detail
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_social.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: components["requestBodies"]["NoticeId"];
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            pics?: unknown;
                            /** @constant */
                            returnCode: "1111";
                            text?: unknown;
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/notices/list": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * notices/list
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_social.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        page?: unknown;
                        pageSize?: unknown;
                        title?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            page?: unknown;
                            pageSize?: unknown;
                            /** @constant */
                            returnCode: "1111";
                            texts?: unknown;
                            total?: unknown;
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/notices/update": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * notices/update
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_social.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        fileListName?: string[];
                        textId?: unknown;
                        textMain?: unknown;
                        title?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                200: components["responses"]["Success"];
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/notifications/search": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * notifications/search
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handlers.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        after?: number;
                        familyId?: number;
                    };
                };
            };
            responses: {
                200: components["responses"]["ItemsSuccess"];
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/position/select": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * position/select
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_resident_devices.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: components["requestBodies"]["NamePage"];
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            page?: unknown;
                            pageSize?: unknown;
                            positionDtos?: unknown;
                            /** @constant */
                            returnCode: "1111";
                            total?: unknown;
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/position/selectFamily": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * position/selectFamily
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_resident_devices.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: components["requestBodies"]["ResidentId"];
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            positionDtos?: unknown;
                            /** @constant */
                            returnCode: "1111";
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/residents/create": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * residents/create
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_resident_devices.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        bothdate?: unknown;
                        customerId?: unknown;
                        deviceId?: unknown;
                        drag?: unknown;
                        foodHate?: unknown;
                        foodLike?: unknown;
                        illness?: unknown;
                        name?: unknown;
                        phone?: unknown;
                        room?: unknown;
                        sex?: unknown;
                        warn?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                200: components["responses"]["Success"];
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/residents/delete": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * residents/delete
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_resident_devices.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: components["requestBodies"]["ResidentId"];
            responses: {
                200: components["responses"]["Success"];
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/residents/details": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * residents/details
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_resident_devices.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: components["requestBodies"]["NamePage"];
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            oldPersonDetails?: unknown;
                            page?: unknown;
                            pageSize?: unknown;
                            /** @constant */
                            returnCode: "1111";
                            total?: unknown;
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/residents/list": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * residents/list
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_resident_devices.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: components["requestBodies"]["NamePage"];
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            oldPeople?: unknown;
                            page?: unknown;
                            pageSize?: unknown;
                            /** @constant */
                            returnCode: "1111";
                            total?: unknown;
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/residents/update": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * residents/update
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_resident_devices.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        bothdate?: unknown;
                        customerId?: unknown;
                        customerName?: unknown;
                        deviceId?: unknown;
                        drag?: unknown;
                        foodHate?: unknown;
                        foodLike?: unknown;
                        illness?: unknown;
                        name?: unknown;
                        oldId?: unknown;
                        phone?: unknown;
                        room?: unknown;
                        sex?: unknown;
                        warn?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                200: components["responses"]["Success"];
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/rule-policies": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * rule-policies
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handlers.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        deviceId?: number;
                        policy?: Record<string, never>;
                        revision?: number;
                    };
                };
            };
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            policy?: unknown;
                            /** @constant */
                            returnCode: "1111";
                            revision?: unknown;
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/security/factor": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * security/factor
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handlers.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        action?: string;
                        otp?: string;
                        password?: string;
                    };
                };
            };
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            recoveryCodes?: unknown;
                            reloginRequired?: unknown;
                            /** @constant */
                            returnCode: "1111";
                            secret?: unknown;
                            uri?: unknown;
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/security/login-audit": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * security/login-audit
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handlers.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        after?: number;
                    };
                };
            };
            responses: {
                200: components["responses"]["ItemsSuccess"];
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/service/status": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /** service/status */
        get: {
            parameters: {
                query?: never;
                header?: never;
                path?: never;
                cookie?: never;
            };
            requestBody?: never;
            responses: {
                /** @description Service database status */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            analytics?: unknown;
                            configError?: unknown;
                            databaseReady?: unknown;
                            primaryEngine?: unknown;
                            /** @constant */
                            returnCode: "1111";
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/setup": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * setup
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_auth.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header?: never;
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        name?: unknown;
                        password?: unknown;
                        phone?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                200: components["responses"]["Success"];
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/user/select": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * user/select
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_user.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        userPhone?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            /** @constant */
                            returnCode: "1111";
                            usersByPhone2?: unknown;
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/vitals/alerts/count": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * vitals/alerts/count
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_vitals.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: components["requestBodies"]["JsonObject"];
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            preTotal?: unknown;
                            rateTotal?: unknown;
                            /** @constant */
                            returnCode: "1111";
                            total?: unknown;
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/vitals/alerts/detail": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * vitals/alerts/detail
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_vitals.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        kind?: unknown;
                        oldId?: unknown;
                        recordId?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            advice?: unknown;
                            bp?: unknown;
                            detail?: unknown;
                            deviceId?: unknown;
                            hasAlert?: unknown;
                            hr?: unknown;
                            message?: unknown;
                            occurTime?: unknown;
                            oldId?: unknown;
                            oldPersonName?: unknown;
                            preBad?: unknown;
                            rateBad?: unknown;
                            reason?: unknown;
                            recordId?: unknown;
                            /** @constant */
                            returnCode: "1111";
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/vitals/alerts/device-counts": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * vitals/alerts/device-counts
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_vitals.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: components["requestBodies"]["JsonObject"];
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            bloodPressure: number;
                            heartRate: number;
                            /** @constant */
                            returnCode: "1111";
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/vitals/alerts/resolve": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * vitals/alerts/resolve
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_vitals.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        kind?: unknown;
                        recordId?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            message?: unknown;
                            /** @constant */
                            returnCode: "1111";
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/vitals/chart": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * vitals/chart
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_vitals.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: components["requestBodies"]["ResidentId"];
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            bps?: unknown;
                            createTimes?: unknown;
                            hrs?: unknown;
                            /** @constant */
                            returnCode: "1111";
                            time?: unknown;
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/vitals/history": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * vitals/history
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_vitals.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: components["requestBodies"]["ResidentId"];
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            /** @constant */
                            returnCode: "1111";
                            vitalRecords?: unknown;
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/vitals/latest": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * vitals/latest
         * @description Implementation: 智慧养老手表管理系统/backend/internal/handlers/handler_vitals.go. Go API DTOs without explicit schemas remain open objects; consult implementation.
         */
        post: {
            parameters: {
                query?: never;
                header: {
                    "X-CSRF-TOKEN": components["parameters"]["CsrfToken"];
                };
                path?: never;
                cookie?: never;
            };
            requestBody: {
                content: {
                    "application/json": {
                        name?: unknown;
                        page?: unknown;
                        pageSize?: unknown;
                        timeFirst?: unknown;
                        timeLast?: unknown;
                    } & {
                        [key: string]: unknown;
                    };
                };
            };
            responses: {
                /** @description Go JSON response; payload varies by operation. */
                200: {
                    headers: {
                        [name: string]: unknown;
                    };
                    content: {
                        "application/json": {
                            page?: unknown;
                            pageSize?: unknown;
                            /** @constant */
                            returnCode: "1111";
                            total?: unknown;
                            vitalRecords?: unknown;
                        } & {
                            [key: string]: unknown;
                        };
                    };
                };
                400: components["responses"]["ApiError"];
                401: components["responses"]["ApiError"];
                403: components["responses"]["ApiError"];
                404: components["responses"]["ApiError"];
                405: components["responses"]["ApiError"];
                409: components["responses"]["ApiError"];
                413: components["responses"]["ApiError"];
                415: components["responses"]["ApiError"];
                429: components["responses"]["ApiError"];
                500: components["responses"]["ApiError"];
                503: components["responses"]["ApiError"];
            };
        };
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
}
export type webhooks = Record<string, never>;
export interface components {
    schemas: {
        ApiError: {
            /** @constant */
            returnCode: "0000";
            code: string;
            message: string;
            reference?: string;
        } & {
            [key: string]: unknown;
        };
    };
    responses: {
        /** @description API failure; HTTP status identifies the error category */
        ApiError: {
            headers: {
                [name: string]: unknown;
            };
            content: {
                "application/json": components["schemas"]["ApiError"];
            };
        };
        /** @description Go JSON response; payload varies by operation. */
        Success: {
            headers: {
                [name: string]: unknown;
            };
            content: {
                "application/json": {
                    /** @constant */
                    returnCode: "1111";
                } & {
                    [key: string]: unknown;
                };
            };
        };
        /** @description Go JSON response; payload varies by operation. */
        ItemsSuccess: {
            headers: {
                [name: string]: unknown;
            };
            content: {
                "application/json": {
                    /** @constant */
                    returnCode: "1111";
                    items?: unknown;
                } & {
                    [key: string]: unknown;
                };
            };
        };
    };
    parameters: {
        CsrfToken: string;
    };
    requestBodies: {
        JsonObject: {
            content: {
                "application/json": {
                    [key: string]: unknown;
                };
            };
        };
        ResidentId: {
            content: {
                "application/json": {
                    oldId?: unknown;
                } & {
                    [key: string]: unknown;
                };
            };
        };
        DeviceId: {
            content: {
                "application/json": {
                    deviceId?: unknown;
                } & {
                    [key: string]: unknown;
                };
            };
        };
        NoticeId: {
            content: {
                "application/json": {
                    textId?: unknown;
                } & {
                    [key: string]: unknown;
                };
            };
        };
        NamePage: {
            content: {
                "application/json": {
                    name?: unknown;
                    page?: unknown;
                    pageSize?: unknown;
                } & {
                    [key: string]: unknown;
                };
            };
        };
    };
    headers: never;
    pathItems: never;
}
export type $defs = Record<string, never>;
export type operations = Record<string, never>;
