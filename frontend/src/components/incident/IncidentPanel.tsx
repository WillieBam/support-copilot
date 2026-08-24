import React from 'react';
import { AlertTriangle, Plus, Search, Filter, RefreshCw, Loader2, Clock, ChevronRight } from 'lucide-react';
import { useIncidentState } from './useIncidentState';
import { CreateIncidentModal } from './CreateIncidentModal';

interface IncidentPanelProps {
  teamId?: string | null;
  selectedIncidentId?: string | null;
  onSelectIncidentForThread: (incidentId: string) => void;
}

export const IncidentPanel: React.FC<IncidentPanelProps> = ({
  teamId,
  selectedIncidentId,
  onSelectIncidentForThread,
}) => {
  const {
    incidents,
    isLoading,
    searchQuery,
    setSearchQuery,
    showAllStatuses,
    setShowAllStatuses,
    isCreateModalOpen,
    setIsCreateModalOpen,
    refreshIncidents,
  } = useIncidentState(teamId);

  const getStatusBadgeClass = (s: string) => {
    switch (s?.toUpperCase()) {
      case 'OPEN':
        return 'bg-red-500/10 text-red-500 border-red-500/20';
      case 'IN_PROGRESS':
        return 'bg-amber-500/10 text-amber-500 border-amber-500/20';
      case 'RESOLVED':
        return 'bg-emerald-500/10 text-emerald-500 border-emerald-500/20';
      case 'CLOSED':
        return 'bg-gray-500/10 text-gray-400 border-gray-500/20';
      default:
        return 'bg-muted text-muted-foreground border-border';
    }
  };

  const formatDate = (isoString?: string) => {
    if (!isoString) return '';
    try {
      return new Date(isoString).toLocaleDateString('en-US', {
        month: 'short',
        day: 'numeric',
      });
    } catch {
      return '';
    }
  };

  return (
    <div className="flex flex-col h-full w-full bg-transparent overflow-hidden">
      {/* Header & Create Button */}
      <div className="p-4 border-b border-border space-y-3 shrink-0">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <AlertTriangle className="w-5 h-5 text-emerald-500" />
            <h2 className="text-base font-bold tracking-tight text-foreground">Team Incidents</h2>
          </div>
          <div className="flex items-center gap-1.5">
            <button
              onClick={refreshIncidents}
              disabled={isLoading || !teamId}
              className="p-1.5 rounded-xl border border-border text-muted-foreground hover:text-foreground hover:bg-muted transition-colors disabled:opacity-50 cursor-pointer"
              title="Refresh incidents"
            >
              <RefreshCw className={`w-3.5 h-3.5 ${isLoading ? 'animate-spin' : ''}`} />
            </button>
            {teamId && (
              <button
                onClick={() => setIsCreateModalOpen(true)}
                className="flex items-center gap-1.5 bg-emerald-500 hover:bg-emerald-600 text-white text-xs px-3 py-1.5 rounded-xl font-medium transition-colors cursor-pointer shadow-sm"
              >
                <Plus className="w-3.5 h-3.5" /> Create
              </button>
            )}
          </div>
        </div>

        {/* Search Bar & Filter Toggle */}
        <div className="flex items-center gap-2">
          <div className="relative flex-1">
            <Search className="w-3.5 h-3.5 absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Search incidents..."
              className="w-full bg-background border border-border rounded-xl pl-8 pr-3 py-1.5 text-xs focus:outline-none focus:ring-2 focus:ring-emerald-500/50"
            />
          </div>
          <button
            onClick={() => setShowAllStatuses(!showAllStatuses)}
            className={`flex items-center gap-1 px-2.5 py-1.5 text-xs border rounded-xl transition-colors cursor-pointer whitespace-nowrap ${
              showAllStatuses
                ? 'bg-emerald-500/10 border-emerald-500/30 text-emerald-500'
                : 'bg-background border-border text-muted-foreground hover:text-foreground'
            }`}
            title={showAllStatuses ? 'Showing All (incl. Resolved/Closed)' : 'Showing Active Only'}
          >
            <Filter className="w-3 h-3" />
            {showAllStatuses ? 'All' : 'Active'}
          </button>
        </div>
      </div>

      {/* Incident List */}
      <div className="flex-1 overflow-y-auto p-3 space-y-2.5">
        {!teamId ? (
          <div className="flex flex-col items-center justify-center h-full text-center p-6 text-muted-foreground">
            <AlertTriangle className="w-8 h-8 text-muted-foreground/40 mb-2" />
            <p className="text-xs">Select a team to view team incidents.</p>
          </div>
        ) : isLoading ? (
          <div className="flex flex-col items-center justify-center h-full py-12 text-muted-foreground">
            <Loader2 className="w-6 h-6 animate-spin text-emerald-500 mb-2" />
            <p className="text-xs">Loading incidents...</p>
          </div>
        ) : incidents.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-full text-center p-6 text-muted-foreground">
            <AlertTriangle className="w-8 h-8 text-muted-foreground/30 mb-2" />
            <p className="text-xs font-medium text-foreground">No incidents found</p>
            <p className="text-[11px] mt-1 text-muted-foreground">
              {showAllStatuses
                ? 'No incidents exist for this team.'
                : 'No active (OPEN/IN_PROGRESS) incidents. Click "All" filter to view resolved/closed incidents.'}
            </p>
          </div>
        ) : (
          incidents.map((inc) => {
            const isSelected = selectedIncidentId === inc.id;

            return (
              <div
                key={inc.id}
                onClick={() => onSelectIncidentForThread(inc.id)}
                className={`p-3.5 rounded-xl border transition-all flex flex-col gap-2 group cursor-pointer ${
                  isSelected
                    ? 'bg-emerald-500/10 border-emerald-500/40 text-foreground'
                    : 'bg-card/70 hover:bg-muted/50 border-border'
                }`}
              >
                <div className="flex items-start justify-between gap-2">
                  <div className="flex items-center gap-1.5">
                    <span className="px-2 py-0.5 text-[10px] font-mono font-bold rounded-md bg-emerald-500/10 text-emerald-500 border border-emerald-500/20">
                      {inc.incident_number || `INC-${inc.id.slice(0, 6)}`}
                    </span>
                    <span className={`px-2 py-0.5 text-[10px] font-semibold rounded-full border ${getStatusBadgeClass(inc.status)}`}>
                      {inc.status}
                    </span>
                  </div>
                  <span className="text-[10px] text-muted-foreground shrink-0 flex items-center gap-1">
                    <Clock className="w-3 h-3" /> {formatDate(inc.history && inc.history.length > 0 ? inc.history[0].updated_at : inc.assigned_at)}
                  </span>

                </div>

                <h3 className="text-xs font-semibold text-foreground group-hover:text-emerald-500 transition-colors line-clamp-2">
                  {inc.title}
                </h3>

                {inc.details && (
                  <p className="text-[11px] text-muted-foreground line-clamp-2">
                    {inc.details}
                  </p>
                )}

                <div className="flex items-center justify-end pt-1 border-t border-border/50 text-[10px]">
                  <span className="flex items-center gap-0.5 text-muted-foreground group-hover:text-emerald-500 font-medium transition-colors text-[11px]">
                    Open thread <ChevronRight className="w-3 h-3" />
                  </span>
                </div>
              </div>
            );
          })
        )}
      </div>

      {/* Create Modal */}
      {isCreateModalOpen && teamId && (
        <CreateIncidentModal
          teamId={teamId}
          onSuccess={refreshIncidents}
          onClose={() => setIsCreateModalOpen(false)}
        />
      )}
    </div>
  );
};
