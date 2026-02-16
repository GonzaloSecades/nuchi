import { formatCurrency, formatPercentage } from '@/lib/utils';
import {
  PolarAngleAxis,
  RadialBar,
  RadialBarChart,
  ResponsiveContainer,
} from 'recharts';
const COLORS = ['#0062FF', '#12C6FF', '#FF647F', '#FF9354'];

type Props = {
  data: {
    name: string;
    value: number;
  }[];
};

export const RadialVariant = ({ data }: Props) => {
  const total = data.reduce((acc, item) => acc + item.value, 0);
  const chartData = data
    .map((item, index) => ({
      name: item.name,
      value: item.value,
      color: COLORS[index % COLORS.length],
      percentage: total > 0 ? (item.value / total) * 100 : 0,
    }))
    .sort((a, b) => b.percentage - a.percentage);

  return (
    <div className="grid grid-cols-1 gap-4 lg:grid-cols-2 lg:items-center">
      <div className="h-75">
        <ResponsiveContainer width="100%" height="100%">
          <RadialBarChart
            cx="50%"
            cy="50%"
            barSize={14}
            innerRadius="20%"
            outerRadius="92%"
            startAngle={90}
            endAngle={-270}
            data={chartData.map((item) => ({
              ...item,
              fill: item.color,
            }))}
          >
            <PolarAngleAxis type="number" domain={[0, 100]} tick={false} />
            <RadialBar background dataKey="percentage" cornerRadius={6} />
          </RadialBarChart>
        </ResponsiveContainer>
      </div>
      <ul className="space-y-2">
        {chartData.map((entry, index) => (
          <li key={`item-${index}`} className="flex items-center space-x-2">
            <span
              className="size-2 rounded-full"
              style={{ backgroundColor: entry.color }}
            />
            <div className="space-x-1">
              <span className="text-muted-foreground text-sm">
                {entry.name}
              </span>
              <span className="text-sm">{formatCurrency(entry.value)}</span>
              <span className="text-muted-foreground text-sm">
                ({formatPercentage(entry.percentage)})
              </span>
            </div>
          </li>
        ))}
      </ul>
    </div>
  );
};
