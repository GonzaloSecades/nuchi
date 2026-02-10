import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { AreaChart, BarChart3, FileSearch, LineChart } from 'lucide-react';
import { useState } from 'react';
import { AreaVariant } from './area-variant';
import { BarVariant } from './bar-variant';
import { LineVariant } from './line-variant';

type Props = {
  data?: {
    date: string;
    income: number;
    expenses: number;
  }[];
};

enum ChartEnum {
  AREA = 'area',
  BAR = 'bar',
  LINE = 'line',
}

export const Chart = ({ data = [] }: Props) => {
  const [chartType, setChartType] = useState(ChartEnum.AREA);

  const onChartTypeChange = (type: ChartEnum) => {
    //TODO: Add paywall;
    setChartType(type);
  };

  return (
    <Card className="border-none drop-shadow-sm">
      <CardHeader className="flex justify-between space-y-2 lg:flex-row lg:items-center lg:space-y-0">
        <CardTitle className="line-clamp-1 text-xl">Transactions</CardTitle>
        <Select defaultValue={chartType} onValueChange={onChartTypeChange}>
          <SelectTrigger className="h-9 rounded-md px-3 lg:w-auto">
            <SelectValue placeholder="Chart type" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={ChartEnum.AREA}>
              <div className="flex items-center">
                <AreaChart className="mr-2 size-4 shrink-0" />
                <p className="line-clamp-1 capitalize">
                  {ChartEnum.AREA} chart
                </p>
              </div>
            </SelectItem>
            <SelectItem value={ChartEnum.LINE}>
              <div className="flex items-center">
                <LineChart className="mr-2 size-4 shrink-0" />
                <p className="line-clamp-1 capitalize">
                  {ChartEnum.LINE} chart
                </p>
              </div>
            </SelectItem>
            <SelectItem value={ChartEnum.BAR}>
              <div className="flex items-center">
                <BarChart3 className="mr-2 size-4 shrink-0" />
                <p className="line-clamp-1 capitalize">{ChartEnum.BAR} chart</p>
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
            {chartType === ChartEnum.AREA && <AreaVariant data={data} />}
            {chartType === ChartEnum.BAR && <BarVariant data={data} />}
            {chartType === ChartEnum.LINE && <LineVariant data={data} />}
          </>
        )}
      </CardContent>
    </Card>
  );
};
