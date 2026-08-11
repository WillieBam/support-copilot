import { useState } from 'react';
import { useAnalyticsState } from './useAnalyticsState';
import { KPICards } from './KPICards';
import { IncidentTrendChart } from './IncidentTrendChart';
import { SLAScatterPlot } from './SLAScatterPlot';
import { ArrowLeft, BarChart3, RefreshCw, AlertCircle, Trash2, Shield, Loader2 } from 'lucide-react';
import { TeamSelector } from '../team/TeamSelector';
import { useTeam } from '@/context/TeamContext';
import { deleteTeam } from '@/service/team/teamService';

interface AnalyticsDashboardViewProps {
  teamId?: string | null;
  onClose: () => void;
}

// AnalyticsDashboardView renders a full-screen view overlay for incident analytics and sla metrics
export function AnalyticsDashboardView({ teamId, onClose }: AnalyticsDashboardViewProps) {
  const { isSuperAdmin, activeMembership, reloadTeams } = useTeam();
  const {
    isAllTeamsMode,
    timeframe,
    setTimeframe,
    slaTarget,
    setSlaTarget,
    pivotedTrend,
    mttr,
    breached,
    isLoading,
    error,
    refreshAnalytics,
  } = useAnalyticsState(teamId, isSuperAdmin);

  const [isDeleting, setIsDeleting] = useState(false);
  const [showConfirmDelete, setShowConfirmDelete] = useState(false);
  const [deleteError, setDeleteError] = useState<string | null>(null);

  const activeTeamName = activeMembership?.team?.team_name || 'Selected Team';
  const activeTeamIdForDelete = activeMembership?.team_id || teamId;

  const handleDeleteTeam = async () => {
    if (!activeTeamIdForDelete) return;
    setIsDeleting(true);
    setDeleteError(null);
    try {
      await deleteTeam(activeTeamIdForDelete);
      setShowConfirmDelete(false);
      await reloadTeams();
    } catch (err: any) {
      setDeleteError(err?.response?.data?.error || 'Failed to delete team');
    } finally {
      setIsDeleting(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 bg-background text-foreground flex flex-col overflow-hidden animate-in fade-in zoom-in-95 duration-200">
      {/* Dashboard Fullscreen Top Bar */}
      <header className="flex items-center justify-between px-6 h-[73px] bg-card border-b border-border shrink-0 z-10 sticky top-0 shadow-sm">
        <div className="flex items-center gap-4">
          <button
            onClick={onClose}
            className="flex items-center gap-2 bg-muted/50 hover:bg-muted border border-border rounded-xl px-3.5 py-1.5 text-foreground text-xs font-semibold transition-all cursor-pointer group"
          >
            <ArrowLeft className="w-4 h-4 text-muted-foreground group-hover:-translate-x-0.5 transition-transform" />
            <span>Back to Workspace</span>
          </button>

          <div className="h-5 w-px bg-border hidden sm:block" />

          <div className="flex items-center gap-2.5">
            <div className="w-8 h-8 rounded-xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-500">
              <BarChart3 className="w-4.5 h-4.5" />
            </div>
            <div>
              <h1 className="text-base font-extrabold text-foreground tracking-tight leading-none flex items-center gap-2">
                Incident Analytics & SLA
                {isSuperAdmin && (
                  <span className="flex items-center gap-1 text-[10px] font-bold text-amber-500 bg-amber-500/10 border border-amber-500/20 px-2 py-0.5 rounded-full uppercase tracking-wider">
                    <Shield className="w-3 h-3" /> Super Admin
                  </span>
                )}
              </h1>
              <p className="text-[11px] text-muted-foreground mt-0.5 hidden md:block">
                Real-time MTTR, incident volume trends, and SLA breach metrics
              </p>
            </div>
          </div>
        </div>

        {/* Control Action Bar */}
        <div className="flex items-center gap-3">
          <TeamSelector />

          {/* Delete Team button next to team selector for super_admin */}
          {isSuperAdmin && activeTeamIdForDelete && (
            <button
              onClick={() => setShowConfirmDelete(true)}
              className="flex items-center gap-1.5 bg-rose-500/10 hover:bg-rose-500/20 border border-rose-500/30 text-rose-500 rounded-xl px-3 py-1.5 text-xs font-medium transition-colors cursor-pointer"
              title={`Delete team ${activeTeamName}`}
            >
              <Trash2 className="w-3.5 h-3.5" />
              <span className="hidden sm:inline">Delete Team</span>
            </button>
          )}

          <button
            onClick={refreshAnalytics}
            disabled={isLoading}
            className="flex items-center justify-center w-9 h-9 bg-transparent border border-border rounded-xl text-muted-foreground hover:text-foreground hover:bg-muted transition-colors disabled:opacity-50 cursor-pointer"
            title="Refresh analytics data"
          >
            <RefreshCw className={`w-4 h-4 ${isLoading ? 'animate-spin text-emerald-500' : ''}`} />
          </button>
        </div>
      </header>

      {/* Main Fullscreen Dashboard Content */}
      <main className="flex-1 overflow-y-auto p-6 md:p-8 space-y-6 w-full max-w-7xl mx-auto">
        {/* All-teams aggregated banner for super admin */}
        {isAllTeamsMode && (
          <div className="bg-amber-500/10 border border-amber-500/20 rounded-2xl p-4 flex items-center gap-3 text-amber-500 text-xs">
            <Shield className="w-4 h-4 shrink-0" />
            <div>
              <span className="font-semibold">All Teams Overview Mode:</span> Displaying combined aggregate analytics across all teams
            </div>
          </div>
        )}

        {/* Error Alert */}
        {error && (
          <div className="bg-rose-500/10 border border-rose-500/20 rounded-2xl p-4 flex items-center justify-between text-rose-500 text-xs">
            <div className="flex items-center gap-2">
              <AlertCircle className="w-4 h-4 shrink-0" />
              <span>{error}</span>
            </div>
            <button
              onClick={refreshAnalytics}
              className="underline font-semibold hover:text-rose-400 cursor-pointer"
            >
              Retry
            </button>
          </div>
        )}

        {/* KPI Cards Row */}
        <KPICards mttr={mttr} isLoading={isLoading} />

        {/* Charts Grid */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 w-full">
          <IncidentTrendChart
            data={pivotedTrend}
            timeframe={timeframe}
            onTimeframeChange={setTimeframe}
            isLoading={isLoading}
          />
          <SLAScatterPlot
            data={breached}
            slaTarget={slaTarget}
            onSlaTargetChange={setSlaTarget}
            isLoading={isLoading}
          />
        </div>
      </main>

      {/* Confirmation Modal for Delete Team */}
      {showConfirmDelete && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4">
          <div className="bg-card border border-border rounded-2xl p-6 w-full max-w-md shadow-2xl space-y-4">
            <div className="flex items-center gap-3 text-rose-500">
              <div className="p-2.5 rounded-xl bg-rose-500/10 border border-rose-500/20">
                <Trash2 className="w-5 h-5" />
              </div>
              <h2 className="text-lg font-bold text-foreground">Delete Team</h2>
            </div>

            <p className="text-xs text-muted-foreground leading-relaxed">
              Are you sure you want to delete team <strong className="text-foreground">{activeTeamName}</strong>? All associated incidents, runbooks, and chat records will be permanently removed.
            </p>

            {deleteError && (
              <div className="p-3 rounded-xl bg-rose-500/10 border border-rose-500/20 text-rose-500 text-xs font-medium">
                {deleteError}
              </div>
            )}

            <div className="flex justify-end gap-3 pt-2">
              <button
                type="button"
                onClick={() => {
                  setShowConfirmDelete(false);
                  setDeleteError(null);
                }}
                disabled={isDeleting}
                className="px-4 py-2 rounded-xl text-xs font-medium text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={handleDeleteTeam}
                disabled={isDeleting}
                className="flex items-center gap-2 bg-rose-500 hover:bg-rose-600 text-white font-medium px-4 py-2 rounded-xl text-xs transition-colors disabled:opacity-50 cursor-pointer shadow-sm"
              >
                {isDeleting ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : 'Confirm Delete'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
