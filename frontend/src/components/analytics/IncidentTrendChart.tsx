import type { PivotedTrendPoint, Timeframe } from './useAnalyticsState';
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts';
import { Calendar } from 'lucide-react';

interface IncidentTrendChartProps {
  data: PivotedTrendPoint[];
  timeframe: Timeframe;
  onTimeframeChange: (tf: Timeframe) => void;
  isLoading: boolean;
}

// CustomTooltip renders a formatted tooltip for the incident trend chart
function CustomTooltip({ active, payload, label }: any) {
  if (active && payload && payload.length) {
    const total = payload.reduce((sum: number, entry: any) => sum + (entry.value || 0), 0);
    return (
      <div className="bg-card/95 border border-border rounded-xl p-3 shadow-xl backdrop-blur-md text-xs">
        <p className="font-bold text-foreground mb-2">{label}</p>
        <div className="flex flex-col gap-1.5 min-w-[140px]">
          {payload.map((entry: any, index: number) => (
            <div key={`item-${index}`} className="flex items-center justify-between gap-4">
              <div className="flex items-center gap-1.5">
                <div
                  className="w-2.5 h-2.5 rounded-full"
                  style={{ backgroundColor: entry.color }}
                />
                <span className="text-muted-foreground font-medium">{entry.name}</span>
              </div>
              <span className="font-bold text-foreground">{entry.value}</span>
            </div>
          ))}
          <div className="border-t border-border pt-1.5 mt-1 flex items-center justify-between font-bold">
            <span className="text-muted-foreground">Total</span>
            <span className="text-foreground">{total}</span>
          </div>
        </div>
      </div>
    );
  }
  return null;
}

// IncidentTrendChart renders a stacked bar chart showing incident volume and status breakdown over time
export function IncidentTrendChart({
  data,
  timeframe,
  onTimeframeChange,
  isLoading,
}: IncidentTrendChartProps) {
  const timeframes: { label: string; value: Timeframe }[] = [
    { label: 'Day', value: 'day' },
    { label: 'Month', value: 'month' },
    { label: 'Year', value: 'year' },
  ];

  return (
    <div className="bg-card/70 border border-border/80 rounded-2xl p-6 backdrop-blur-md shadow-sm flex flex-col w-full h-full min-h-[380px]">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 mb-4">
        <div>
          <h3 className="text-base font-bold text-foreground tracking-tight">Incident Volume & Trend</h3>
          <p className="text-xs text-muted-foreground mt-0.5">Aggregated frequency by status across time buckets</p>
        </div>

        {/* Timeframe Filter Pills */}
        <div className="flex items-center gap-1.5 bg-muted/40 border border-border p-1 rounded-xl shrink-0">
          <div className="flex items-center gap-1 px-2 text-xs text-muted-foreground font-medium">
            <Calendar className="w-3.5 h-3.5 text-emerald-500" />
            <span className="hidden sm:inline">Timeframe:</span>
          </div>
          {timeframes.map((tf) => (
            <button
              key={tf.value}
              onClick={() => onTimeframeChange(tf.value)}
              className={`px-2.5 py-1 rounded-lg text-xs font-semibold transition-all cursor-pointer ${
                timeframe === tf.value
                  ? 'bg-card text-emerald-500 shadow-sm border border-border'
                  : 'text-muted-foreground hover:text-foreground hover:bg-muted/50'
              }`}
            >
              {tf.label}
            </button>
          ))}
        </div>
      </div>

      <div className="flex-1 w-full min-h-[300px] flex items-center justify-center relative">
        {isLoading ? (
          <div className="w-full h-[280px] bg-muted/30 rounded-xl animate-pulse flex items-center justify-center">
            <span className="text-xs text-muted-foreground">Loading trend analytics...</span>
          </div>
        ) : !data || data.length === 0 ? (
          <div className="w-full h-[280px] border border-dashed border-border rounded-xl flex items-center justify-center text-muted-foreground text-xs">
            No incident trend data available for this timeframe
          </div>
        ) : (
          <ResponsiveContainer width="100%" height={300}>
            <BarChart data={data} margin={{ top: 10, right: 10, left: -20, bottom: 0 }}>
              <CartesianGrid strokeDasharray="3 3" opacity={0.15} vertical={false} />
              <XAxis
                dataKey="time_bucket"
                tick={{ fill: 'currentColor', fontSize: 11 }}
                axisLine={{ opacity: 0.2 }}
                tickLine={false}
              />
              <YAxis
                allowDecimals={false}
                tick={{ fill: 'currentColor', fontSize: 11 }}
                axisLine={{ opacity: 0.2 }}
                tickLine={false}
              />
              <Tooltip content={<CustomTooltip />} />
              <Legend
                verticalAlign="top"
                align="right"
                iconType="circle"
                wrapperStyle={{ paddingBottom: '16px', fontSize: '12px' }}
              />
              <Bar dataKey="OPEN" name="Open" stackId="a" fill="#ef4444" radius={[0, 0, 0, 0]} />
              <Bar dataKey="IN_PROGRESS" name="In Progress" stackId="a" fill="#f59e0b" radius={[0, 0, 0, 0]} />
              <Bar dataKey="RESOLVED" name="Resolved" stackId="a" fill="#10b981" radius={[0, 0, 0, 0]} />
              <Bar dataKey="CLOSED" name="Closed" stackId="a" fill="#6b7280" radius={[4, 4, 0, 0]} />
            </BarChart>
          </ResponsiveContainer>
        )}
      </div>
    </div>
  );
}
