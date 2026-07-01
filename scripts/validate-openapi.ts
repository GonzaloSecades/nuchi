import { readFileSync } from 'node:fs';

const HTTP_METHODS = new Set([
  'get',
  'put',
  'post',
  'delete',
  'options',
  'head',
  'patch',
  'trace',
]);

type JsonObject = Record<string, unknown>;

function fail(message: string): never {
  throw new Error(message);
}

function asObject(value: unknown, label: string): JsonObject {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    fail(`${label} must be an object`);
  }

  return value as JsonObject;
}

function stringField(object: JsonObject, key: string, label: string) {
  const value = object[key];

  if (typeof value !== 'string' || value.length === 0) {
    fail(`${label}.${key} must be a non-empty string`);
  }

  return value;
}

function validateOperation(path: string, method: string, value: unknown) {
  const operation = asObject(value, `${path}.${method}`);
  stringField(operation, 'operationId', `${path}.${method}`);

  const responses = asObject(operation.responses, `${path}.${method}.responses`);

  if (Object.keys(responses).length === 0) {
    fail(`${path}.${method}.responses must define at least one response`);
  }
}

function main() {
  const specPath = process.argv[2] ?? 'openapi/nuchi.openapi.json';
  const document = asObject(
    JSON.parse(readFileSync(specPath, 'utf8')),
    specPath
  );

  const openapi = stringField(document, 'openapi', specPath);
  if (!/^3\.(0|1)\.\d+$/.test(openapi)) {
    fail(`${specPath}.openapi must be 3.0.x or 3.1.x`);
  }

  const info = asObject(document.info, `${specPath}.info`);
  stringField(info, 'title', `${specPath}.info`);
  stringField(info, 'version', `${specPath}.info`);

  const paths = asObject(document.paths, `${specPath}.paths`);
  let operationCount = 0;

  for (const [path, pathItem] of Object.entries(paths)) {
    if (!path.startsWith('/')) {
      fail(`OpenAPI path must start with "/": ${path}`);
    }

    const pathObject = asObject(pathItem, path);

    for (const [method, operation] of Object.entries(pathObject)) {
      if (!HTTP_METHODS.has(method)) {
        continue;
      }

      validateOperation(path, method, operation);
      operationCount += 1;
    }
  }

  if (operationCount === 0) {
    fail(`${specPath}.paths must define at least one operation`);
  }

  console.log(
    `Validated ${specPath}: OpenAPI ${openapi}, ${operationCount} operation(s)`
  );
}

try {
  main();
} catch (error) {
  console.error(error instanceof Error ? error.message : error);
  process.exit(1);
}
