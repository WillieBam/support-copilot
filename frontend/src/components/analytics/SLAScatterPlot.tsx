import type { BreachedIncident } from '@/types/analytics';
import type { SLAFilterOption } from './useAnalyticsState';
import {
  ScatterChart,
  Scatter,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ReferenceLine,
  ResponsiveContainer,
} from 'recharts';
import { Filter } from 'lucide-react';

interface SLAScatterPlotProps {
  data: BreachedIncident[];
  slaTarget: SLAFilterOption;
  onSlaTargetChange: (target: SLAFilterOption) => void;
  isLoading: boolean;
}

// CustomScatterTooltip formats scatter plot dot hover information
function CustomScatterTooltip({ active, payload }: any) {
  if (active && payload && payload.length) {
    const item = payload[0].payload;
    const duration = Math.round((item.duration_minutes || 0) * 10) / 10;
    const createdDate = item.created_at ? new Date(item.created_at).toLocaleString() : 'N/A';
    const resolvedDate = item.resolved_at ? new Date(item.resolved_at).toLocaleString() : 'N/A';

    return (
      <div className="bg-card/95 border border-border rounded-xl p-3 shadow-xl backdrop-blur-md text-xs max-w-xs">
        <p className="font-bold text-foreground truncate mb-1">{item.title || 'Untitled Incident'}</p>
        <div className="space-y-1 text-muted-foreground">
          <div className="flex justify-between gap-3">
            <span>Duration:</span>
            <span className="font-semibold text-rose-500">{duration} mins</span>
          </div>
          <div className="flex justify-between gap-3 text-[11px]">
            <span>Created:</span>
            <span className="text-foreground">{createdDate}</span>
          </div>
          <div className="flex justify-between gap-3 text-[11px]">
            <span>Resolved:</span>
            <span className="text-foreground">{resolvedDate}</span>
          </div>
        </div>
      </div>
    );
  }
  return null;
}

// SLAScatterPlot renders resolution duration outliers against SLA target threshold
export function SLAScatterPlot({
  data,
  slaTarget,
  onSlaTargetChange,
  isLoading,
}: SLAScatterPlotProps) {
  const filterOptions: { label: string; value: SLAFilterOption }[] = [
    { label: '0m (All)', value: 0 },
    { label: '15m+', value: 15 },
    { label: '30m+', value: 30 },
    { label: '60m+', value: 60 },
  ];

  // transform data points for ScatterChart
  const scatterData = data.map((item, index) => ({
    x: index + 1,
    y: Math.round(item.duration_minutes * 10) / 10,
    title: item.title,
    created_at: item.created_at,
    resolved_at: item.resolved_at,
    duration_minutes: item.duration_minutes,
  }));

  const effectiveSlaTarget = slaTarget === 0 ? 30 : slaTarget;

  return (
    <div className="bg-card/70 border border-border/80 rounded-2xl p-6 backdrop-blur-md shadow-sm flex flex-col w-full h-full min-h-[380px]">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 mb-4">
        <div>
          <h3 className="text-base font-bold text-foreground tracking-tight">Resolution Time Outliers vs SLA</h3>
          <p className="text-xs text-muted-foreground mt-0.5">Individual incident durations filtered by SLA breach threshold</p>
        </div>

        {/* SLA Filter Pills */}
        <div className="flex items-center gap-1.5 bg-muted/40 border border-border p-1 rounded-xl shrink-0">
          <div className="flex items-center gap-1 px-2 text-xs text-muted-foreground font-medium">
            <Filter className="w-3.5 h-3.5 text-emerald-500" />
            <span className="hidden sm:inline">Min Threshold:</span>
          </div>
          {filterOptions.map((opt) => (
            <button
              key={opt.value}
              onClick={() => onSlaTargetChange(opt.value)}
              className={`px-2.5 py-1 rounded-lg text-xs font-semibold transition-all cursor-pointer ${
                slaTarget === opt.value
                  ? 'bg-card text-emerald-500 shadow-sm border border-border'
                  : 'text-muted-foreground hover:text-foreground hover:bg-muted/50'
              }`}
            >
              {opt.label}
            </button>
          ))}
        </div>
      </div>

      <div className="flex-1 w-full min-h-[300px] flex items-center justify-center relative">
        {isLoading ? (
          <div className="w-full h-[280px] bg-muted/30 rounded-xl animate-pulse flex items-center justify-center">
            <span className="text-xs text-muted-foreground">Loading scatter resolution data...</span>
          </div>
        ) : !data || data.length === 0 ? (
          <div className="w-full h-[280px] border border-dashed border-border rounded-xl flex items-center justify-center text-muted-foreground text-xs">
            No incidents exceeding {slaTarget}m resolution threshold
          </div>
        ) : (
          <ResponsiveContainer width="100%" height={300}>
            <ScatterChart margin={{ top: 20, right: 20, left: -10, bottom: 10 }}>
              <CartesianGrid strokeDasharray="3 3" opacity={0.15} />
              <XAxis
                type="number"
                dataKey="x"
                name="Incident #"
                tick={{ fill: 'currentColor', fontSize: 11 }}
                axisLine={{ opacity: 0.2 }}
                tickLine={false}
                label={{ value: 'Incident Index', position: 'bottom', offset: 0, fill: 'currentColor', fontSize: 11, opacity: 0.6 }}
              />
              <YAxis
                type="number"
                dataKey="y"
                name="Duration (mins)"
                unit="m"
                tick={{ fill: 'currentColor', fontSize: 11 }}
                axisLine={{ opacity: 0.2 }}
                tickLine={false}
                label={{ value: 'Duration (mins)', angle: -90, position: 'insideLeft', offset: 15, fill: 'currentColor', fontSize: 11, opacity: 0.6 }}
              />
              <Tooltip content={<CustomScatterTooltip />} />
              <ReferenceLine
                y={effectiveSlaTarget}
                stroke="#f43f5e"
                strokeDasharray="4 4"
                label={{
                  value: `SLA Target (${effectiveSlaTarget}m)`,
                  fill: '#f43f5e',
                  fontSize: 11,
                  position: 'top',
                }}
              />
              <Scatter name="Breached Incident" data={scatterData} fill="#10b981" />
            </ScatterChart>
          </ResponsiveContainer>
        )}
      </div>
    </div>
  );
}
