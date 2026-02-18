'use client';

import { useMemo } from 'react';
import { SingleValue } from 'react-select';
import CreateableSelect from 'react-select/creatable';

type Props = {
  onChange: (value?: string) => void;
  onCreate?: (value: string) => void;
  options?: { label: string; value: string }[];
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
  const onSelect = (option: SingleValue<{ label: string; value: string }>) => {
    onChange(option?.value);
  };

  const formattedValue = useMemo(() => {
    if (value === undefined) {
      return undefined;
    }
    return options.find((option) => option.value === value) || null;
  }, [options, value]);

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
      options={options}
      onCreateOption={onCreate}
      isDisabled={disabled}
      placeholder={placeholder}
    />
  );
};
