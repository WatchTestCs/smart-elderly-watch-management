// Generated from the native Go OpenAPI. Do not edit.
export interface paths {
  "/api/v1/health/latest": {
    post: {
      requestBody: { content: { "application/json": { "oldId"?: number; } } };
      responses: { 200: { content: { "application/json": { "items": ({ "bindingVersion": number; "deviceId": number | null; "oldId": number; "revision": number; "sample": { "deleted"?: boolean; "end": string; "groupId"?: string; "metadata"?: { [key: string]: string; }; "metric": string; "recordId": string; "semantics": string; "source": string; "sourceDevice": string; "sourceVersion"?: string; "stage"?: string; "start": string; "timeBasis"?: string; "unit": string; "value": number | null; "version": number; }; "sampleId": string; })[]; "oldId": number; "returnCode": "1111"; } } } };
    };
  };
}
