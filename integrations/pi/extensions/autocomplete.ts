import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { createRunner, type RootlineError } from "../runner";

/**
 * Tool input for autocomplete suggestions.
 */
interface AutocompleteToolInput {
  /** Directory path to get autocomplete suggestions for (required) */
  path: string;
  /** Optional field name prefix to match against (e.g., "est" to match "estado") */
  field?: string;
  /** Disable autocomplete if noisy or slow (optional, default false) */
  disabled?: boolean;
}

/**
 * Schema field definition with type and enum values.
 */
interface FieldDefinition {
  /** Field name */
  field: string;
  /** Field type (string, number, boolean, array, object, etc.) */
  type: string;
  /** Allowed enum values if field is an enum type */
  values?: string[];
  /** Whether the field is required */
  required?: boolean;
  /** Description or documentation for the field */
  description?: string;
}

/**
 * Autocomplete suggestions result.
 */
interface AutocompleteSuggestions {
  /** All available field names in the schema */
  fields: string[];
  /** Detailed field definitions with types and enum values */
  suggestions: FieldDefinition[];
  /** Whether autocomplete is disabled */
  disabled: boolean;
  /** Error message if applicable */
  error?: string;
}

/**
 * Describe result from rootline (subset for autocomplete).
 */
interface DescribeResult {
  schema?: Record<string, any>;
  applies?: string[];
}

/**
 * Registers the rootline-autocomplete tool for schema field and record reference suggestions.
 *
 * Used for IDE autocomplete and context injection in Pi agents.
 * Queries the effective schema for a directory and returns:
 * - List of available field names
 * - Field definitions with types and enum values
 * - Optional filtering by field name prefix
 *
 * Can be disabled via `disabled` parameter for noisy or slow scenarios.
 *
 * Timeout: 5s (safe for editor suggestion lists).
 *
 * @param pi ExtensionAPI instance
 */
export default function (pi: ExtensionAPI) {
  const run = createRunner(pi);

  pi.registerTool({
    name: "rootline-autocomplete",
    description:
      "Get autocomplete suggestions for schema fields and enum values in a directory. " +
      "Returns available field names, their types, and enum values if applicable. " +
      "Can be disabled if noisy. Useful for IDE autocomplete and context injection. " +
      "Fast operation (<5s timeout).",
    input_schema: {
      type: "object" as const,
      properties: {
        path: {
          type: "string",
          description:
            "Directory path to get autocomplete suggestions for (required)",
        },
        field: {
          type: "string",
          description:
            "Optional field name prefix to match against (e.g., 'est' to match 'estado')",
        },
        disabled: {
          type: "boolean",
          description:
            "Disable autocomplete if noisy or slow (optional, default false)",
        },
      },
      required: ["path"],
    },
    execute: async (input: unknown) => {
      const { path, field, disabled } = input as AutocompleteToolInput;

      // Initialize result
      const result: AutocompleteSuggestions = {
        fields: [],
        suggestions: [],
        disabled: disabled ?? false,
      };

      // Early exit if disabled
      if (disabled) {
        return result;
      }

      // Call rootline describe to get schema
      const describeResult = await run<DescribeResult>(
        ["describe", path, "--output", "json"],
        { cwd: path, timeout: 5000 }
      );

      if (!describeResult.ok) {
        result.error = describeResult.error.message;
        return result;
      }

      const schema = describeResult.data.schema || {};

      // Extract field names and build suggestions
      const fieldNames = Object.keys(schema);

      // Filter by field prefix if provided
      const filteredFields = field
        ? fieldNames.filter((name) =>
            name.toLowerCase().startsWith(field.toLowerCase())
          )
        : fieldNames;

      // Build suggestion objects
      const suggestions: FieldDefinition[] = filteredFields.map((fieldName) => {
        const fieldDef = schema[fieldName];
        const suggestion: FieldDefinition = {
          field: fieldName,
          type: fieldDef.type || "unknown",
        };

        // Extract enum values if present
        if (fieldDef.enum && Array.isArray(fieldDef.enum)) {
          suggestion.values = fieldDef.enum;
        }

        // Extract required flag if present
        if (fieldDef.required !== undefined) {
          suggestion.required = fieldDef.required;
        }

        // Extract description if present
        if (fieldDef.description) {
          suggestion.description = fieldDef.description;
        }

        return suggestion;
      });

      result.fields = filteredFields;
      result.suggestions = suggestions;

      return result;
    },
  });
}
