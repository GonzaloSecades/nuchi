export type SelectOption = {
  label: string;
  value: string;
};

export function mergeCreatedSelectOption(
  options: SelectOption[],
  createdOption: SelectOption | null
) {
  if (!createdOption) {
    return options;
  }

  if (options.some((option) => option.value === createdOption.value)) {
    return options;
  }

  return [createdOption, ...options];
}
