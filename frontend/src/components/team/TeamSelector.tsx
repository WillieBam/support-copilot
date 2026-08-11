import { ChevronDown, Users, Plus } from 'lucide-react';
import { useTeamSelectorState } from '@/components/team/TeamSelectorState';
import { TeamMembersModal } from '@/components/team/TeamMembersModal';
import { CreateTeamModal } from '@/components/team/CreateTeamModal';

export const TeamSelector = () => {
  const {
    memberships,
    activeMembership,
    isOwner,
    isOpen,
    isMembersModalOpen,
    isCreateModalOpen,
    toggleDropdown,
    handleSelectTeam,
    openMembersModal,
    closeMembersModal,
    openCreateModal,
    closeCreateModal,
    reloadTeams,
  } = useTeamSelectorState();

  if (memberships.length === 0) {
    return (
      <div className="relative inline-block text-left">
        <button
          onClick={toggleDropdown}
          className="flex items-center gap-2 bg-card/60 border border-border rounded-[20px] px-4 py-1.5 text-foreground hover:bg-muted transition-colors text-sm cursor-pointer shadow-sm"
        >
          <Users className="w-4 h-4 text-emerald-500" />
          <span className="font-medium truncate max-w-[140px]">No Team</span>
          <ChevronDown className="w-4 h-4 text-muted-foreground" />
        </button>

        {isOpen && (
          <div className="absolute right-0 mt-2 w-64 rounded-xl border border-border bg-card shadow-xl z-50 p-4 backdrop-blur-xl text-center">
            <p className="text-xs text-muted-foreground mb-1">No team joined yet</p>
            <p className="text-[11px] text-muted-foreground/70 mb-3">Create or join a team to manage members.</p>
            <button
              onClick={openCreateModal}
              className="w-full flex items-center justify-center gap-2 bg-emerald-500 hover:bg-emerald-600 text-white px-4 py-2 rounded-xl transition-colors text-sm font-medium cursor-pointer shadow-sm"
            >
              <Plus className="w-4 h-4" />
              Create Team
            </button>
          </div>
        )}
      </div>
    );
  }

  const getRoleBadge = (item?: (typeof memberships)[0] | null) => {
    if (!item) return '';
    if (item.team_id === '') return 'OVERVIEW';
    if (item.id?.startsWith('virtual-') || item.role === 'admin') return 'SUPER ADMIN';
    return item.role;
  };

  return (
    <div className="relative inline-block text-left">
      {/* active team dropdown toggle */}
      <button
        onClick={toggleDropdown}
        className="flex items-center gap-2 bg-card/60 border border-border rounded-[20px] px-4 py-1.5 text-foreground hover:bg-muted transition-colors text-sm cursor-pointer shadow-sm"
      >
        <Users className="w-4 h-4 text-emerald-500" />
        <span className="font-medium truncate max-w-[140px]">
          {activeMembership?.team?.team_name || 'Select Team'}
        </span>
        <span className="text-[10px] uppercase font-bold px-1.5 py-0.5 rounded bg-muted text-muted-foreground">
          {getRoleBadge(activeMembership)}
        </span>
        <ChevronDown className="w-4 h-4 text-muted-foreground" />
      </button>

      {/* dropdown popup */}
      {isOpen && (
        <div className="absolute right-0 mt-2 w-64 rounded-xl border border-border bg-card shadow-xl z-50 py-2 backdrop-blur-xl">
          <div className="px-3 py-1.5 text-[11px] font-bold text-muted-foreground uppercase tracking-wider">
            Available Teams
          </div>

          <div className="max-h-48 overflow-y-auto">
            {memberships.map((item) => (
              <button
                key={item.team_id}
                onClick={() => handleSelectTeam(item.team_id)}
                className={`w-full flex items-center justify-between px-3 py-2 text-sm transition-colors text-left ${
                  item.team_id === activeMembership?.team_id
                    ? 'bg-emerald-500/10 text-emerald-500 font-medium'
                    : 'text-foreground hover:bg-muted'
                }`}
              >
                <span className="truncate">{item.team.team_name}</span>
                <span className="text-[10px] uppercase font-semibold px-1.5 py-0.5 rounded border border-border text-muted-foreground">
                  {getRoleBadge(item)}
                </span>
              </button>
            ))}
          </div>

          <div className="border-t border-border mt-2 pt-2 px-2 space-y-1">
            {Boolean(activeMembership?.team_id) && (
              <button
                onClick={openMembersModal}
                className="w-full flex items-center gap-2 px-3 py-1.5 text-xs text-emerald-500 hover:bg-emerald-500/10 rounded-lg transition-colors font-medium cursor-pointer"
              >
                <Users className="w-3.5 h-3.5" />
                Manage Team Members
              </button>
            )}
            <button
              onClick={openCreateModal}
              className="w-full flex items-center gap-2 px-3 py-1.5 text-xs text-emerald-500 hover:bg-emerald-500/10 rounded-lg transition-colors font-medium cursor-pointer"
            >
              <Plus className="w-3.5 h-3.5" />
              Create Team
            </button>
          </div>
        </div>
      )}

      {/* team members modal */}
      {isMembersModalOpen && (
        <TeamMembersModal
          teamId={activeMembership?.team_id || ''}
          teamName={activeMembership?.team?.team_name || ''}
          isOwner={isOwner}
          onClose={closeMembersModal}
        />
      )}

      {/* create team modal */}
      {isCreateModalOpen && (
        <CreateTeamModal
          reloadTeams={reloadTeams}
          onClose={closeCreateModal}
        />
      )}
    </div>
  );
};