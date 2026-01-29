import { test as base, request } from '@playwright/test';

/**
 * Custom fixture that provides helpers for interacting with the Apparatus API
 */
export const test = base.extend({
  apparatusAPI: async ({}, use) => {
    const baseURL = 'http://localhost:8080';
    const apiContext = await request.newContext({ baseURL });

    const api = {
      /**
       * Create a new run
       * @param {string} name - Run name
       * @param {string} [parentRunUuid] - Optional parent run UUID for nested runs
       * @returns {Promise<string>} Run UUID
       */
      async createRun(name, parentRunUuid = null) {
        let url = `/api/runs?name=${encodeURIComponent(name)}`;
        if (parentRunUuid) {
          url += `&parent_run_uuid=${encodeURIComponent(parentRunUuid)}`;
        }
        const response = await apiContext.get(url);
        if (!response.ok()) {
          throw new Error(`Failed to create run: ${response.status()} ${await response.text()}`);
        }
        const data = await response.json();
        return data.id;
      },

      /**
       * Log a parameter
       * @param {string} runUuid - Run UUID
       * @param {string} key - Parameter key
       * @param {string|number|boolean} value - Parameter value
       * @param {string} type - Parameter type: 'string', 'int', 'float', 'bool'
       */
      async logParam(runUuid, key, value, type = 'string') {
        const params = new URLSearchParams({
          run_uuid: runUuid,
          key,
          value: String(value),
          type
        });
        const response = await apiContext.get(`/api/params?${params}`);
        if (!response.ok()) {
          throw new Error(`Failed to log param: ${response.status()} ${await response.text()}`);
        }
        return await response.json();
      },

      /**
       * Log a metric
       * @param {string} runUuid - Run UUID
       * @param {string} key - Metric key
       * @param {number} xValue - X-axis value (step/time)
       * @param {number} yValue - Y-axis value (metric value)
       */
      async logMetric(runUuid, key, xValue, yValue) {
        const body = {
          run_uuid: runUuid,
          key,
          values: [{ x_value: xValue, y_value: yValue }],
          logged_at_epoch_millis: Date.now()
        };
        const response = await apiContext.post('/api/metrics', {
          data: body
        });
        if (!response.ok()) {
          throw new Error(`Failed to log metric: ${response.status()} ${await response.text()}`);
        }
        return await response.json();
      },

      /**
       * Log an artifact (file upload)
       * @param {string} runUuid - Run UUID
       * @param {string} path - Artifact path (e.g., "plots/loss.png")
       * @param {Buffer|string} fileContent - File content as Buffer or string
       * @param {string} fileName - File name for multipart upload
       */
      async logArtifact(runUuid, path, fileContent, fileName = 'file.txt') {
        const formData = {
          run_uuid: runUuid,
          path,
          file: {
            name: fileName,
            mimeType: 'application/octet-stream',
            buffer: Buffer.isBuffer(fileContent) ? fileContent : Buffer.from(fileContent)
          }
        };
        const response = await apiContext.post('/api/artifacts', {
          multipart: formData
        });
        if (!response.ok()) {
          throw new Error(`Failed to log artifact: ${response.status()} ${await response.text()}`);
        }
        return await response.json();
      },

      /**
       * Helper: start a run and log params/metrics
       * @param {object} options - {name, params, metrics}
       * @returns {Promise<string>} Run UUID
       */
      async startRun({ name, params = {}, metrics = {} }) {
        const runId = await this.createRun(name);

        // Log parameters
        for (const [key, value] of Object.entries(params)) {
          let type = 'string';
          if (typeof value === 'number') {
            type = Number.isInteger(value) ? 'int' : 'float';
          } else if (typeof value === 'boolean') {
            type = 'bool';
          }
          await this.logParam(runId, key, value, type);
        }

        // Log metrics
        for (const [key, value] of Object.entries(metrics)) {
          await this.logMetric(runId, key, value);
        }

        return runId;
      }
    };

    await use(api);
    await apiContext.dispose();
  },
});

export { expect } from '@playwright/test';
