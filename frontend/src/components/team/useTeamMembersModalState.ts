import { useState, useEffect, useCallback, type FormEvent } from 'react';
import { addTeamMember, removeTeamMember, fetchTeamMembers, searchUsers, deactivateUser } from '@/service/team/teamService';
import type { TeamMember } from '@/types/team';
import type { UserSearchResult } from '@/types/user';

export const useTeamMembersModalState = (teamId: string) => {
  const [members, setMembers] = useState<TeamMember[]>([]);
  const [isLoadingMembers, setIsLoadingMembers] = useState<boolean>(true);
  const [deletingUserId, setDeletingUserId] = useState<string | null>(null);
  const [deactivatingUserId, setDeactivatingUserId] = useState<string | null>(null);

  const [searchQuery, setSearchQuery] = useState<string>('');
  const [searchResults, setSearchResults] = useState<UserSearchResult[]>([]);
  const [selectedUser, setSelectedUser] = useState<UserSearchResult | null>(null);
  const [isSearching, setIsSearching] = useState<boolean>(false);
  const [isSubmitting, setIsSubmitting] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);

  const loadMembers = useCallback(async () => {
    if (!teamId) return;
    setIsLoadingMembers(true);
    try {
      const data = await fetchTeamMembers(teamId);
      setMembers(data || []);
    } catch (err: any) {
      console.error('Failed to load team members:', err);
    } finally {
      setIsLoadingMembers(false);
    }
  }, [teamId]);

  useEffect(() => {
    loadMembers();
  }, [loadMembers]);

  useEffect(() => {
    if (searchQuery.trim().length < 2) {
      setSearchResults([]);
      return;
    }

    const timer = setTimeout(async () => {
      setIsSearching(true);
      try {
        const results = await searchUsers(searchQuery.trim());
        setSearchResults(results || []);
      } catch (err: any) {
        console.error('Failed to search users:', err);
        setSearchResults([]);
      } finally {
        setIsSearching(false);
      }
    }, 300);

    return () => clearTimeout(timer);
  }, [searchQuery]);

  const handleSelectUser = (user: UserSearchResult) => {
    setSelectedUser(user);
    setSearchQuery('');
    setSearchResults([]);
    setError(null);
  };

  const handleClearSelection = () => {
    setSelectedUser(null);
  };

  const handleAddMember = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    if (!selectedUser) {
      setError('Please search and select a user to add');
      return;
    }

    setIsSubmitting(true);
    setError(null);
    setSuccessMsg(null);

    try {
      await addTeamMember(teamId, { user_id: selectedUser.id });
      setSuccessMsg('Added member successfully');
      setSelectedUser(null);
      await loadMembers();
    } catch (err: any) {
      setError(err?.response?.data?.error || 'failed to add team member');
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleDeleteMember = async (targetUserId: string) => {
    setDeletingUserId(targetUserId);
    setError(null);
    setSuccessMsg(null);

    try {
      await removeTeamMember(teamId, targetUserId);
      setSuccessMsg('Removed member successfully');
      await loadMembers();
    } catch (err: any) {
      setError(err?.response?.data?.error || 'failed to remove team member');
    } finally {
      setDeletingUserId(null);
    }
  };

  const handleDeactivateUser = async (targetUserId: string) => {
    if (!window.confirm('Are you sure you want to deactivate this user account? The user will immediately be disabled from logging in.')) {
      return;
    }
    setDeactivatingUserId(targetUserId);
    setError(null);
    setSuccessMsg(null);

    try {
      await deactivateUser(targetUserId);
      setSuccessMsg('User account deactivated successfully');
      await loadMembers();
    } catch (err: any) {
      setError(err?.response?.data?.error || 'failed to deactivate user account');
    } finally {
      setDeactivatingUserId(null);
    }
  };

  return {
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
    refreshMembers: loadMembers,
  };
};
