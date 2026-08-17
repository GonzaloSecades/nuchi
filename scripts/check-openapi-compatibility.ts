import { execFileSync } from 'node:child_process';
import { readFileSync } from 'node:fs';

type JsonObject = Record<string, unknown>;

const HTTP_METHODS = [
  'get',
  'put',
  'post',
  'delete',
  'patch',
  'head',
  'options',
  'trace',
];

function object(value: unknown): JsonObject {
  return value && typeof value === 'object' && !Array.isArray(value)
    ? (value as JsonObject)
    : {};
}

function strings(value: unknown): string[] {
  return Array.isArray(value)
    ? value.filter((item): item is string => typeof item === 'string')
    : [];
}

function loadBase(ref: string, path: string): JsonObject {
  const text = execFileSync('git', ['show', `${ref}:${path}`], {
    encoding: 'utf8',
  });
  return object(JSON.parse(text));
}

function checkSchemas(
  base: JsonObject,
  current: JsonObject,
  failures: string[]
) {
  for (const [name, baseValue] of Object.entries(base)) {
    const currentValue = current[name];
    if (!currentValue) {
      failures.push(`removed schema ${name}`);
      continue;
    }
    const baseSchema = object(baseValue);
    const currentSchema = object(currentValue);
    const baseProperties = object(baseSchema.properties);
    const currentProperties = object(currentSchema.properties);

    for (const property of Object.keys(baseProperties)) {
      if (!(property in currentProperties))
        failures.push(`removed property ${name}.${property}`);
    }
    const baseRequired = new Set(strings(baseSchema.required));
    for (const property of strings(currentSchema.required)) {
      if (!baseRequired.has(property))
        failures.push(`made property required ${name}.${property}`);
    }
    const currentEnum = new Set(strings(currentSchema.enum));
    for (const value of strings(baseSchema.enum)) {
      if (!currentEnum.has(value))
        failures.push(`removed enum value ${name}.${value}`);
    }
  }
}

function checkParameters(
  base: JsonObject,
  current: JsonObject,
  failures: string[]
) {
  for (const [name, baseValue] of Object.entries(base)) {
    const currentValue = current[name];
    if (!currentValue) {
      failures.push(`removed parameter ${name}`);
      continue;
    }
    if (
      object(baseValue).required !== true &&
      object(currentValue).required === true
    ) {
      failures.push(`made parameter required ${name}`);
    }
  }
}

function checkOperations(
  base: JsonObject,
  current: JsonObject,
  failures: string[]
) {
  for (const [path, basePathValue] of Object.entries(base)) {
    const currentPathValue = current[path];
    if (!currentPathValue) {
      failures.push(`removed path ${path}`);
      continue;
    }
    const basePath = object(basePathValue);
    const currentPath = object(currentPathValue);
    for (const method of HTTP_METHODS) {
      if (!basePath[method]) continue;
      if (!currentPath[method]) {
        failures.push(`removed operation ${method.toUpperCase()} ${path}`);
        continue;
      }
      const baseOperation = object(basePath[method]);
      const currentOperation = object(currentPath[method]);
      const currentResponses = object(currentOperation.responses);
      for (const status of Object.keys(object(baseOperation.responses))) {
        if (!(status in currentResponses)) {
          failures.push(
            `removed response ${status} from ${method.toUpperCase()} ${path}`
          );
        }
      }
      if (
        object(baseOperation.requestBody).required !== true &&
        object(currentOperation.requestBody).required === true
      ) {
        failures.push(
          `made request body required for ${method.toUpperCase()} ${path}`
        );
      }
    }
  }
}

function main() {
  const baseRef = process.argv[2];
  if (!baseRef)
    throw new Error(
      'usage: check-openapi-compatibility.ts <base-git-ref> [spec-path]'
    );
  const path = process.argv[3] ?? 'openapi/nuchi.openapi.json';
  const base = loadBase(baseRef, path);
  const current = object(JSON.parse(readFileSync(path, 'utf8')));
  const failures: string[] = [];

  checkOperations(object(base.paths), object(current.paths), failures);
  const baseComponents = object(base.components);
  const currentComponents = object(current.components);
  checkSchemas(
    object(baseComponents.schemas),
    object(currentComponents.schemas),
    failures
  );
  checkParameters(
    object(baseComponents.parameters),
    object(currentComponents.parameters),
    failures
  );

  if (failures.length > 0) {
    throw new Error(
      `Potentially breaking OpenAPI changes:\n- ${failures.join('\n- ')}`
    );
  }
  console.log(`OpenAPI compatibility check passed against ${baseRef}`);
}

try {
  main();
} catch (error) {
  console.error(error instanceof Error ? error.message : error);
  process.exit(1);
}
