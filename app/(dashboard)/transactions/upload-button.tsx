import { Button } from '@/components/ui/button';
import type { CSVUploadResults } from '@/lib/transaction-import';
import { Upload } from 'lucide-react';
import type { HTMLAttributes } from 'react';
import { useCSVReader } from 'react-papaparse';

type Props = {
  onUpload: (results: CSVUploadResults) => void;
};

type CSVReaderRenderProps = {
  getRootProps: () => HTMLAttributes<HTMLElement>;
};

export const UploadButton = ({ onUpload }: Props) => {
  const { CSVReader } = useCSVReader();
  return (
    <CSVReader config={{ dynamicTyping: false }} onUploadAccepted={onUpload}>
      {({ getRootProps }: CSVReaderRenderProps) => (
        <Button size="sm" className="w-full lg:w-auto" {...getRootProps()}>
          <Upload className="mr-2 size-4" />
          Import
        </Button>
      )}
    </CSVReader>
  );
};
