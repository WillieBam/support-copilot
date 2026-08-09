import { useState, useEffect, useCallback, useRef } from 'react';
import { fetchUserTeams, fetchTeamMembers } from '@/service/team/teamService';
import type { UserMembership, TeamRole, TeamMember } from '@/types/team';

// localStorage key for tracking which team IDs this user has already seen
const knownTeamsKey = (uid: string) => `knownTeams:${uid}`;

export const useTeamState = (isSignedIn: boolean, userEmail?: string) => {
  const [memberships, setMemberships] = useState<UserMembership[]>([]);
  const [activeTeamId, setActiveTeamId] = useState<string | null>(null);
  const [teamMembers, setTeamMembers] = useState<TeamMember[]>([]);
  const [isLoading, setIsLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);

  // New-team notification state
  const [newTeamNames, setNewTeamNames] = useState<string[]>([]);
  // Guard so detection only runs once per load cycle
  const hasDetected = useRef(false);

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

      // ── New-team notification detection ──────────────────────────
      // Only run once per sign-in session to avoid repeat toasts
      if (!hasDetected.current && userEmail) {
        hasDetected.current = true;
        const key = knownTeamsKey(userEmail);
        const stored: string[] = JSON.parse(localStorage.getItem(key) ?? '[]');
        const storedSet = new Set(stored);
        const currentIds = userMemberships.map((m) => m.team_id);

        // Find teams the user is now in but weren't in the snapshot
        const newOnes = userMemberships.filter(
          (m) => !storedSet.has(m.team_id) && m.role !== 'owner'
        );

        if (newOnes.length > 0) {
          setNewTeamNames(newOnes.map((m) => m.team.team_name));
        }

        // Always update the snapshot to current full list
        localStorage.setItem(key, JSON.stringify(currentIds));
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

  // getUserDisplayName resolves a user_id or raw email string to a display-friendly name
  const getUserDisplayName = (val?: string): string => {
    if (!val) return 'System';

    // search teamMembers directly - list is small so linear scan is fine
    for (const m of teamMembers) {
      const u = m.user;
      if (!u) continue;
      if (m.user_id === val || u.id === val) {
        if (u.display_name?.trim()) return u.display_name.trim();
        if (u.email?.trim()) return u.email.split('@')[0];
      }
    }

    // If val is an email string
    if (val.includes('@')) return val.split('@')[0];

    return val;
  };

  const activeMembership = memberships.find((m) => m.team_id === activeTeamId) || null;
  const activeRole: TeamRole | null = activeMembership ? activeMembership.role : null;
  const isOwner = activeRole === 'owner';

  const selectTeam = (teamId: string) => {
    setActiveTeamId(teamId);
  };

  /** Called by the toast once the user has seen the notification */
  const dismissNewTeams = useCallback(() => {
    setNewTeamNames([]);
  }, []);

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
    newTeamNames,
    dismissNewTeams,
  };
};