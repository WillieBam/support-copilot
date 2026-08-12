import { X, UserPlus, Loader2, Search, Check, Trash2, Users, Shield, UserX } from 'lucide-react';
import { useTeamMembersModalState } from './useTeamMembersModalState';

interface TeamMembersModalProps {
  teamId: string;
  teamName: string;
  isOwner: boolean;
  isSuperAdmin?: boolean;
  onClose: () => void;
}

export const TeamMembersModal = ({ teamId, teamName, isOwner, isSuperAdmin, onClose }: TeamMembersModalProps) => {
  const {
    members,
    isLoadingMembers,
    deletingUserId,
    deactivatingUserId,
    searchQuery,
    setSearchQuery,
    searchResults,
    selectedUser,
    isSearching,
    isSubmitting,
    error,
    successMsg,
    handleSelectUser,
    handleClearSelection,
    handleAddMember,
    handleDeleteMember,
    handleDeactivateUser,
  } = useTeamMembersModalState(teamId);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4">
      <div className="bg-card border border-border rounded-[20px] p-6 w-full max-w-lg shadow-2xl relative max-h-[90vh] flex flex-col">
        <button
          onClick={onClose}
          className="absolute top-4 right-4 text-muted-foreground hover:text-foreground p-1 transition-colors cursor-pointer"
        >
          <X className="w-5 h-5" />
        </button>

        {/* Modal Header */}
        <div className="flex items-center gap-3 mb-4 shrink-0">
          <div className="p-2.5 rounded-xl bg-emerald-500/10 text-emerald-500">
            <Users className="w-5 h-5" />
          </div>
          <div>
            <h2 className="text-lg font-bold text-foreground">Team Members</h2>
            <p className="text-xs text-muted-foreground">
              Team: <span className="font-medium text-foreground">{teamName}</span> ({members.length} {members.length === 1 ? 'member' : 'members'})
            </p>
          </div>
        </div>

        {error && (
          <div className="mb-4 p-3 rounded-xl bg-red-500/10 border border-red-500/20 text-red-500 text-xs font-medium shrink-0">
            {error}
          </div>
        )}

        {successMsg && (
          <div className="mb-4 p-3 rounded-xl bg-emerald-500/10 border border-emerald-500/20 text-emerald-500 text-xs font-medium shrink-0">
            {successMsg}
          </div>
        )}

        {/* Add Member Section (Only for Owner) */}
        {isOwner && (
          <div className="mb-5 pb-5 border-b border-border shrink-0">
            <h3 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-2 flex items-center gap-1.5">
              <UserPlus className="w-3.5 h-3.5 text-emerald-500" /> Add New Member
            </h3>

            <form onSubmit={handleAddMember} className="space-y-3">
              {selectedUser ? (
                <div className="flex items-center justify-between p-3 rounded-xl bg-emerald-500/10 border border-emerald-500/30">
                  <div className="flex items-center gap-2">
                    <Check className="w-4 h-4 text-emerald-500" />
                    <div>
                      <p className="text-xs font-medium text-foreground">
                        {selectedUser.display_name || selectedUser.email}
                      </p>
                      <p className="text-[11px] text-muted-foreground">{selectedUser.email}</p>
                    </div>
                  </div>
                  <button
                    type="button"
                    onClick={handleClearSelection}
                    className="text-muted-foreground hover:text-foreground p-1 cursor-pointer"
                  >
                    <X className="w-4 h-4" />
                  </button>
                </div>
              ) : (
                <div className="relative">
                  <div className="relative flex items-center">
                    <Search className="w-4 h-4 absolute left-3 text-muted-foreground pointer-events-none" />
                    <input
                      type="text"
                      value={searchQuery}
                      onChange={(e) => setSearchQuery(e.target.value)}
                      placeholder="Search user by email or name..."
                      className="w-full bg-muted border border-border rounded-xl pl-9 pr-9 py-2 text-sm text-foreground focus:outline-none focus:border-emerald-500 transition-colors"
                    />
                    {isSearching && (
                      <Loader2 className="w-4 h-4 absolute right-3 text-emerald-500 animate-spin" />
                    )}
                  </div>

                  {/* Dropdown Search Results */}
                  {searchResults.length > 0 && (
                    <div className="absolute left-0 right-0 mt-1.5 max-h-40 overflow-y-auto rounded-xl border border-border bg-card shadow-xl z-50 py-1">
                      {searchResults.map((user) => (
                        <button
                          key={user.id}
                          type="button"
                          onClick={() => handleSelectUser(user)}
                          className="w-full text-left px-3 py-2 hover:bg-muted transition-colors flex flex-col cursor-pointer"
                        >
                          <span className="text-xs font-medium text-foreground">
                            {user.display_name || user.email}
                          </span>
                          <span className="text-[11px] text-muted-foreground">{user.email}</span>
                        </button>
                      ))}
                    </div>
                  )}
                </div>
              )}

              <div className="flex justify-end">
                <button
                  type="submit"
                  disabled={isSubmitting || !selectedUser}
                  className="flex items-center gap-2 bg-emerald-500 hover:bg-emerald-600 text-white font-medium px-4 py-1.5 rounded-xl text-xs transition-colors disabled:opacity-50 cursor-pointer shadow-sm"
                >
                  {isSubmitting ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : 'Add to Team'}
                </button>
              </div>
            </form>
          </div>
        )}

        {/* Member List */}
        <div className="flex-1 overflow-y-auto min-h-[160px] pr-1">
          <h3 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-3">
            Current Members
          </h3>

          {isLoadingMembers ? (
            <div className="flex items-center justify-center py-8 text-muted-foreground">
              <Loader2 className="w-5 h-5 animate-spin text-emerald-500 mr-2" />
              <span className="text-xs">Loading members...</span>
            </div>
          ) : members.length === 0 ? (
            <div className="text-center py-8 text-muted-foreground text-xs">
              No members found in this team.
            </div>
          ) : (
            <div className="space-y-2">
              {members.map((member) => {
                const userEmail = member.user?.email || member.user_id;
                const displayName = member.user?.display_name || userEmail.split('@')[0];
                const initial = displayName.charAt(0).toUpperCase();
                const isMemberOwner = member.role === 'owner';
                const isDeleting = deletingUserId === member.user_id;

                return (
                  <div
                    key={member.id}
                    className="flex items-center justify-between p-3 rounded-xl bg-card border border-border hover:bg-muted/40 transition-colors"
                  >
                    <div className="flex items-center gap-3 min-w-0">
                      <div className="w-8 h-8 rounded-full bg-muted border border-border flex items-center justify-center text-foreground font-bold text-xs shrink-0 shadow-inner">
                        {initial}
                      </div>
                      <div className="flex flex-col min-w-0">
                        <span className="text-xs font-medium text-foreground truncate">
                          {displayName}
                        </span>
                        <span className="text-[11px] text-muted-foreground truncate">
                          {userEmail}
                        </span>
                      </div>
                    </div>

                    <div className="flex items-center gap-2 shrink-0">
                      {isMemberOwner ? (
                        <span className="flex items-center gap-1 text-[10px] font-semibold px-2 py-0.5 rounded bg-emerald-500/10 text-emerald-500 border border-emerald-500/20">
                          <Shield className="w-3 h-3" /> Owner
                        </span>
                      ) : (
                        <span className="text-[10px] px-2 py-0.5 rounded bg-muted text-muted-foreground border border-border font-medium">
                          Member
                        </span>
                      )}

                      {isOwner && !isMemberOwner && (
                        <button
                          onClick={() => handleDeleteMember(member.user_id)}
                          disabled={isDeleting}
                          className="p-1.5 text-muted-foreground hover:text-red-500 hover:bg-red-500/10 rounded-lg transition-colors cursor-pointer disabled:opacity-50"
                          title="Remove member"
                        >
                          {isDeleting ? (
                            <Loader2 className="w-3.5 h-3.5 animate-spin text-red-500" />
                          ) : (
                            <Trash2 className="w-3.5 h-3.5" />
                          )}
                        </button>
                      )}

                      {isSuperAdmin && (
                        <button
                          onClick={() => handleDeactivateUser(member.user_id)}
                          disabled={deactivatingUserId === member.user_id}
                          className="p-1.5 text-amber-500/80 hover:text-amber-400 hover:bg-amber-500/10 rounded-lg transition-colors cursor-pointer disabled:opacity-50"
                          title="Deactivate User Account (Superadmin)"
                        >
                          {deactivatingUserId === member.user_id ? (
                            <Loader2 className="w-3.5 h-3.5 animate-spin text-amber-500" />
                          ) : (
                            <UserX className="w-3.5 h-3.5" />
                          )}
                        </button>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>

        {/* Modal Footer */}
        <div className="flex justify-end pt-4 mt-2 border-t border-border shrink-0">
          <button
            type="button"
            onClick={onClose}
            className="px-4 py-1.5 text-xs text-muted-foreground hover:text-foreground transition-colors cursor-pointer font-medium"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
};
