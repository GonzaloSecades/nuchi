import { Button } from '@/components/ui/button';
import { Upload } from 'lucide-react';
import { useCSVReader } from 'react-papaparse';

type Props = {
  onUpload: (results: any) => void;
};

type CSVReaderRenderProps = {
  getRootProps: () => React.HTMLAttributes<HTMLElement>;
};

export const UploadButton = ({ onUpload }: Props) => {
  const { CSVReader } = useCSVReader();
  //Todo add a paywall
  return (
    <CSVReader onUploadAccepted={onUpload}>
      {({ getRootProps }: CSVReaderRenderProps) => (
        <Button size="sm" className="w-full lg:w-auto" {...getRootProps()}>
          <Upload className="mr-2 size-4" />
          Import
        </Button>
      )}
    </CSVReader>
  );
};
