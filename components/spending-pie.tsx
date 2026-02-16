import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { FileSearch, Loader2, PieChart, Radar, Target } from 'lucide-react';
import { useState } from 'react';
import { PieVariant } from './pie-variant';
import { RadarVariant } from './radar-variant';
import { RadialVariant } from './radial-variant';
import { Skeleton } from './ui/skeleton';

type Props = {
  data?: {
    name: string;
    value: number;
  }[];
};

enum SpendingPieEnum {
  PIE = 'pie',
  RADAR = 'radar',
  RADIAL = 'radial',
}

export const SpendingPie = ({ data = [] }: Props) => {
  const [chartType, setChartType] = useState(SpendingPieEnum.PIE);

  const onChartTypeChange = (type: SpendingPieEnum) => {
    //TODO: Add paywall;
    setChartType(type);
  };

  return (
    <Card className="border-none drop-shadow-sm">
      <CardHeader className="flex justify-between space-y-2 lg:flex-row lg:items-center lg:space-y-0">
        <CardTitle className="line-clamp-1 text-xl">Categories</CardTitle>
        <Select defaultValue={chartType} onValueChange={onChartTypeChange}>
          <SelectTrigger className="h-9 rounded-md px-3 lg:w-auto">
            <SelectValue placeholder="Chart type" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={SpendingPieEnum.PIE}>
              <div className="flex items-center">
                <PieChart className="mr-2 size-4 shrink-0" />
                <p className="line-clamp-1 capitalize">
                  {SpendingPieEnum.PIE} chart
                </p>
              </div>
            </SelectItem>
            <SelectItem value={SpendingPieEnum.RADAR}>
              <div className="flex items-center">
                <Radar className="mr-2 size-4 shrink-0" />
                <p className="line-clamp-1 capitalize">
                  {SpendingPieEnum.RADAR} chart
                </p>
              </div>
            </SelectItem>
            <SelectItem value={SpendingPieEnum.RADIAL}>
              <div className="flex items-center">
                <Target className="mr-2 size-4 shrink-0" />
                <p className="line-clamp-1 capitalize">
                  {SpendingPieEnum.RADIAL} chart
                </p>
              </div>
            </SelectItem>
          </SelectContent>
        </Select>
      </CardHeader>
      <CardContent>
        {data.length === 0 ? (
          <div className="flex h-87.5 flex-col items-center justify-center gap-y-4">
            <FileSearch className="text-muted-foreground size-6" />
            <p className="text-muted-foreground text-sm">
              No data for this period
            </p>
          </div>
        ) : (
          <>
            {chartType === SpendingPieEnum.PIE && <PieVariant data={data} />}
            {chartType === SpendingPieEnum.RADAR && (
              <RadarVariant data={data} />
            )}
            {chartType === SpendingPieEnum.RADIAL && (
              <RadialVariant data={data} />
            )}
          </>
        )}
      </CardContent>
    </Card>
  );
};

export const SpendingPieLoading = () => {
  return (
    <Card className="border-none drop-shadow-sm">
      <CardHeader className="flex justify-between space-y-2 lg:flex-row lg:items-center lg:space-y-0">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-8 w-full lg:w-30" />
      </CardHeader>
      <CardContent>
        <div className="flex h-87.5 w-full items-center justify-center">
          <Loader2 className="size-6 animate-spin text-slate-300" />
        </div>
      </CardContent>
    </Card>
  );
};
