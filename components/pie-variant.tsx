import { formatPercentage } from '@/lib/utils';
import {
  type PieSectorShapeProps,
  Legend,
  Pie,
  PieChart,
  ResponsiveContainer,
  Sector,
  Tooltip,
} from 'recharts';
import { CategoryTooltip } from './category-tooltip';
const COLORS = ['#0062FF', '#12C6FF', '#FF647F', '#FF9354'];

type Props = {
  data: {
    name: string;
    value: number;
  }[];
};

const PieSliceShape = (props: PieSectorShapeProps) => {
  const index = props.index ?? 0;
  return <Sector {...props} fill={COLORS[index % COLORS.length]} />;
};

export const PieVariant = ({ data }: Props) => {
  const total = data.reduce((acc, item) => acc + item.value, 0);
  const legendEntries = data
    .map((item, index) => ({
      name: item.name,
      value: item.value,
      color: COLORS[index % COLORS.length],
      percentage: total > 0 ? (item.value / total) * 100 : 0,
    }))
    .sort((a, b) => b.percentage - a.percentage);

  return (
    <ResponsiveContainer width="100%" height={300}>
      <PieChart>
        <Legend
          layout="horizontal"
          verticalAlign="bottom"
          align="right"
          iconType="circle"
          content={() => {
            return (
              <ul className="flex flex-col space-y-2">
                {legendEntries.map((entry, index) => {
                  return (
                    <li
                      key={`item-${index}`}
                      className="flex items-center space-x-2"
                    >
                      <span
                        className="size-2 rounded-full"
                        style={{ backgroundColor: entry.color }}
                      />
                      <div className="space-x-1">
                        <span className="text-muted-foreground text-sm">
                          {entry.name}
                        </span>
                        <span className="text-sm">
                          {formatPercentage(entry.percentage)}
                        </span>
                      </div>
                    </li>
                  );
                })}
              </ul>
            );
          }}
        />
        <Tooltip content={<CategoryTooltip />} />
        <Pie
          isAnimationActive={false}
          data={data}
          cx="50%"
          cy="50%"
          outerRadius={90}
          innerRadius={60}
          paddingAngle={2}
          fill="#8884d8"
          dataKey="value"
          labelLine={false}
          shape={PieSliceShape}
        />
      </PieChart>
    </ResponsiveContainer>
  );
};
