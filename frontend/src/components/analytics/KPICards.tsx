import type { MTTRResult } from '@/types/analytics';
import { Clock, AlertTriangle, CheckCircle2, TrendingUp } from 'lucide-react';

interface KPICardsProps {
  mttr: MTTRResult | null;
  isLoading: boolean;
}

// KPICards displays key performance indicators for mean resolution time, breaches, and compliance rate
export function KPICards({ mttr, isLoading }: KPICardsProps) {
  const mttrMinutes = mttr ? Math.round(mttr.mttr_minutes * 10) / 10 : 0;
  const totalResolved = mttr ? mttr.total_resolved : 0;
  const breaches = mttr ? mttr.sla_breaches : 0;
  const compliance = mttr ? Math.round(mttr.compliance_rate * 10) / 10 : 100;

  const isHighCompliance = compliance >= 90;
  const isMediumCompliance = compliance >= 75 && compliance < 90;

  return (
    <div className="grid grid-cols-1 md:grid-cols-4 gap-4 w-full">
      {/* MTTR Card */}
      <div className="bg-card/70 border border-border/80 rounded-2xl p-5 backdrop-blur-md shadow-sm hover:border-emerald-500/30 transition-all duration-300 flex flex-col justify-between relative overflow-hidden group">
        <div className="flex items-center justify-between">
          <span className="text-muted-foreground text-xs font-semibold uppercase tracking-wider">Mean Time To Resolve</span>
          <div className="w-8 h-8 rounded-xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-500">
            <Clock className="w-4 h-4" />
          </div>
        </div>
        <div className="mt-4 flex items-baseline gap-2">
          {isLoading ? (
            <div className="h-9 w-24 bg-muted/50 rounded-lg animate-pulse" />
          ) : (
            <>
              <span className="text-3xl font-extrabold text-foreground tracking-tight">{mttrMinutes}</span>
              <span className="text-xs text-muted-foreground font-medium">mins</span>
            </>
          )}
        </div>
        <div className="mt-2 flex items-center gap-1 text-[11px] text-muted-foreground">
          <TrendingUp className="w-3 h-3 text-emerald-500" />
          <span>Average across {totalResolved} resolved incidents</span>
        </div>
      </div>

      {/* Total Resolved Card */}
      <div className="bg-card/70 border border-border/80 rounded-2xl p-5 backdrop-blur-md shadow-sm hover:border-emerald-500/30 transition-all duration-300 flex flex-col justify-between relative overflow-hidden group">
        <div className="flex items-center justify-between">
          <span className="text-muted-foreground text-xs font-semibold uppercase tracking-wider">Total Resolved</span>
          <div className="w-8 h-8 rounded-xl bg-blue-500/10 border border-blue-500/20 flex items-center justify-center text-blue-500">
            <CheckCircle2 className="w-4 h-4" />
          </div>
        </div>
        <div className="mt-4 flex items-baseline gap-2">
          {isLoading ? (
            <div className="h-9 w-20 bg-muted/50 rounded-lg animate-pulse" />
          ) : (
            <span className="text-3xl font-extrabold text-foreground tracking-tight">{totalResolved}</span>
          )}
        </div>
        <div className="mt-2 text-[11px] text-muted-foreground">
          <span>Completed lifecycle incidents</span>
        </div>
      </div>

      {/* SLA Breaches Card */}
      <div className="bg-card/70 border border-border/80 rounded-2xl p-5 backdrop-blur-md shadow-sm hover:border-amber-500/30 transition-all duration-300 flex flex-col justify-between relative overflow-hidden group">
        <div className="flex items-center justify-between">
          <span className="text-muted-foreground text-xs font-semibold uppercase tracking-wider">SLA Breaches</span>
          <div className="w-8 h-8 rounded-xl bg-rose-500/10 border border-rose-500/20 flex items-center justify-center text-rose-500">
            <AlertTriangle className="w-4 h-4" />
          </div>
        </div>
        <div className="mt-4 flex items-baseline gap-2">
          {isLoading ? (
            <div className="h-9 w-16 bg-muted/50 rounded-lg animate-pulse" />
          ) : (
            <span className={`text-3xl font-extrabold tracking-tight ${breaches > 0 ? 'text-rose-500' : 'text-foreground'}`}>
              {breaches}
            </span>
          )}
        </div>
        <div className="mt-2 text-[11px] text-muted-foreground">
          <span>Exceeded target resolution threshold</span>
        </div>
      </div>

      {/* SLA Compliance Rate Card */}
      <div className="bg-card/70 border border-border/80 rounded-2xl p-5 backdrop-blur-md shadow-sm hover:border-emerald-500/30 transition-all duration-300 flex flex-col justify-between relative overflow-hidden group">
        <div className="flex items-center justify-between">
          <span className="text-muted-foreground text-xs font-semibold uppercase tracking-wider">SLA Compliance</span>
          <div className={`w-8 h-8 rounded-xl flex items-center justify-center border ${
            isHighCompliance
              ? 'bg-emerald-500/10 border-emerald-500/20 text-emerald-500'
              : isMediumCompliance
              ? 'bg-amber-500/10 border-amber-500/20 text-amber-500'
              : 'bg-rose-500/10 border-rose-500/20 text-rose-500'
          }`}>
            <TrendingUp className="w-4 h-4" />
          </div>
        </div>
        <div className="mt-4 flex items-baseline gap-2">
          {isLoading ? (
            <div className="h-9 w-24 bg-muted/50 rounded-lg animate-pulse" />
          ) : (
            <>
              <span className={`text-3xl font-extrabold tracking-tight ${
                isHighCompliance ? 'text-emerald-500' : isMediumCompliance ? 'text-amber-500' : 'text-rose-500'
              }`}>
                {compliance}%
              </span>
            </>
          )}
        </div>
        <div className="mt-2 text-[11px] text-muted-foreground">
          <span>Target compliance rate indicator</span>
        </div>
      </div>
    </div>
  );
}
