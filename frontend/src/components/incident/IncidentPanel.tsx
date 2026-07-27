import React, { useState, useEffect } from 'react';
import { AlertTriangle, Plus, Search, Filter, RefreshCw, Loader2, Clock } from 'lucide-react';
import type { TeamIncident } from '@/types/incident';
import { fetchTeamIncidents } from '@/service/incident/incidentService';
import { IncidentDetailModal } from './IncidentDetailModal';
import { CreateIncidentModal } from './CreateIncidentModal';

interface IncidentPanelProps {
  teamId?: string | null;
}

export const IncidentPanel: React.FC<IncidentPanelProps> = ({ teamId }) => {
  const [incidents, setIncidents] = useState<TeamIncident[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [showAllStatuses, setShowAllStatuses] = useState(false);

  // Modal states
  const [selectedIncidentId, setSelectedIncidentId] = useState<string | null>(null);
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);

  const loadIncidents = async () => {
    if (!teamId) {
      setIncidents([]);
      return;
    }
    setIsLoading(true);
    try {
      const data = await fetchTeamIncidents(teamId);
      setIncidents(data || []);
    } catch {
      setIncidents([]);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    loadIncidents();
  }, [teamId]);

  // Filtering & Search Logic
  const filteredIncidents = incidents.filter((inc) => {
    const statusUpper = inc.status?.toUpperCase() || '';
    // By default, show only active incidents: OPEN, IN_PROGRESS (or IN PRORESS)
    const isActive = statusUpper === 'OPEN' || statusUpper === 'IN_PROGRESS' || statusUpper === 'IN PRORESS';
    
    if (!showAllStatuses && !isActive) {
      return false;
    }

    if (searchQuery.trim()) {
      const q = searchQuery.toLowerCase();
      const matchTitle = inc.title?.toLowerCase().includes(q);
      const matchDetails = inc.details?.toLowerCase().includes(q);
      const matchStatus = inc.status?.toLowerCase().includes(q);
      return matchTitle || matchDetails || matchStatus;
    }

    return true;
  });

  // Limit default active view to top 5 incidents (or all when searching/filtering)
  const displayedIncidents = !showAllStatuses && !searchQuery.trim()
    ? filteredIncidents.slice(0, 5)
    : filteredIncidents;

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
              onClick={loadIncidents}
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
        ) : displayedIncidents.length === 0 ? (
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
          displayedIncidents.map((inc) => (
            <div
              key={inc.id}
              onClick={() => setSelectedIncidentId(inc.id)}
              className="p-3.5 bg-card/70 hover:bg-muted/50 border border-border rounded-xl cursor-pointer transition-all hover:border-emerald-500/30 group"
            >
              <div className="flex items-start justify-between gap-2 mb-1.5">
                <span className={`px-2 py-0.5 text-[10px] font-semibold rounded-full border ${getStatusBadgeClass(inc.status)}`}>
                  {inc.status}
                </span>
                <span className="text-[10px] text-muted-foreground shrink-0 flex items-center gap-1">
                  <Clock className="w-3 h-3" /> {formatDate(inc.assigned_at)}
                </span>
              </div>

              <h3 className="text-xs font-semibold text-foreground group-hover:text-emerald-500 transition-colors line-clamp-2">
                {inc.title}
              </h3>

              {inc.details && (
                <p className="text-[11px] text-muted-foreground line-clamp-2 mt-1">
                  {inc.details}
                </p>
              )}
            </div>
          ))
        )}
      </div>

      {/* Detail / Editing Modal */}
      {selectedIncidentId && (
        <IncidentDetailModal
          incidentId={selectedIncidentId}
          onClose={() => setSelectedIncidentId(null)}
          onUpdated={loadIncidents}
        />
      )}

      {/* Create Modal */}
      {isCreateModalOpen && teamId && (
        <CreateIncidentModal
          teamId={teamId}
          onSuccess={loadIncidents}
          onClose={() => setIsCreateModalOpen(false)}
        />
      )}
    </div>
  );
};
