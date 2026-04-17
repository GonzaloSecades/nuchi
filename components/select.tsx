'use client';

import {
  mergeCreatedSelectOption,
  type SelectOption,
} from '@/lib/select-options';
import { useMemo, useState } from 'react';
import { SingleValue } from 'react-select';
import CreateableSelect from 'react-select/creatable';

type Props = {
  onChange: (value?: string) => void;
  onCreate?: (value: string) => Promise<string | undefined> | string | void;
  options?: SelectOption[];
  value?: string | null | undefined;
  disabled?: boolean;
  placeholder?: string;
};

export const Select = ({
  onChange,
  onCreate,
  options = [],
  value,
  disabled,
  placeholder,
}: Props) => {
  const [createdOption, setCreatedOption] = useState<SelectOption | null>(null);

  const mergedOptions = useMemo(
    () => mergeCreatedSelectOption(options, createdOption),
    [createdOption, options]
  );

  const onSelect = (option: SingleValue<SelectOption>) => {
    onChange(option?.value);
  };

  const onCreateOption = async (label: string) => {
    const createdId = await onCreate?.(label);

    if (!createdId) {
      return;
    }

    const nextCreatedOption = {
      label,
      value: createdId,
    };

    setCreatedOption(nextCreatedOption);
    onChange(createdId);
  };

  const formattedValue = useMemo(() => {
    if (value === undefined) {
      return undefined;
    }
    return mergedOptions.find((option) => option.value === value) || null;
  }, [mergedOptions, value]);

  return (
    <CreateableSelect
      className="h-10 text-base md:text-sm"
      classNamePrefix="nuchi-select"
      styles={{
        control: (base: Record<string, unknown>) => ({
          ...base,
          borderColor: '#e2e8f0',
          minHeight: '2.5rem',
          ':hover': {
            borderColor: '#e2e8f0',
          },
        }),
      }}
      {...(formattedValue === undefined ? {} : { value: formattedValue })}
      onChange={onSelect}
      options={mergedOptions}
      onCreateOption={onCreate ? onCreateOption : undefined}
      isDisabled={disabled}
      placeholder={placeholder}
    />
  );
};
