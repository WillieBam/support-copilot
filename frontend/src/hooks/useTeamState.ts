import { useState, useEffect, useCallback, useMemo } from 'react';
import { fetchUserTeams, fetchTeamMembers } from '@/service/team/teamService';
import type { UserMembership, TeamRole, TeamMember } from '@/types/team';

export const useTeamState = (isSignedIn: boolean) => {
  const [memberships, setMemberships] = useState<UserMembership[]>([]);
  const [activeTeamId, setActiveTeamId] = useState<string | null>(null);
  const [teamMembers, setTeamMembers] = useState<TeamMember[]>([]);
  const [isLoading, setIsLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);

  // load user team memberships
  const loadTeams = useCallback(async () => {
    if (!isSignedIn) return;
    setIsLoading(true);
    setError(null);
    try {
      const data = await fetchUserTeams();
      const userMemberships = data.memberships || [];
      setMemberships(userMemberships);
      
      // default active team to first joined team if not selected
      if (userMemberships.length > 0 && !activeTeamId) {
        setActiveTeamId(userMemberships[0].team_id);
      }
    } catch (err: any) {
      setError(err?.response?.data?.error || 'failed to load user teams');
    } finally {
      setIsLoading(false);
    }
  }, [isSignedIn, activeTeamId]);

  useEffect(() => {
    loadTeams();
  }, [loadTeams]);

  // load team members whenever activeTeamId changes
  useEffect(() => {
    if (!activeTeamId || !isSignedIn) {
      setTeamMembers([]);
      return;
    }
    fetchTeamMembers(activeTeamId)
      .then((data) => setTeamMembers(data || []))
      .catch((err) => console.error('Failed to load active team members:', err));
  }, [activeTeamId, isSignedIn]);

  // map of user_id -> display info
  const userMap = useMemo(() => {
    const map = new Map<string, { display_name?: string; email?: string }>();
    teamMembers.forEach((m) => {
      const u = m.user;
      if (u) {
        if (m.user_id) map.set(m.user_id, u);
        if (u.id) map.set(u.id, u);
      }
    });
    return map;
  }, [teamMembers]);

  const getUserDisplayName = useCallback((val?: string) => {
    if (!val) return 'System';
    
    // Check if val is in userMap
    const found = userMap.get(val);
    if (found) {
      if (found.display_name?.trim()) return found.display_name.trim();
      if (found.email?.trim()) return found.email.split('@')[0];
    }

    // If val is an email string
    if (val.includes('@')) {
      return val.split('@')[0];
    }

    return val;
  }, [userMap]);

  const activeMembership = memberships.find((m) => m.team_id === activeTeamId) || null;
  const activeRole: TeamRole | null = activeMembership ? activeMembership.role : null;
  const isOwner = activeRole === 'owner';

  const selectTeam = (teamId: string) => {
    setActiveTeamId(teamId);
  };

  return {
    memberships,
    activeTeamId,
    activeMembership,
    activeRole,
    isOwner,
    teamMembers,
    getUserDisplayName,
    isLoading,
    error,
    selectTeam,
    reloadTeams: loadTeams,
  };
};