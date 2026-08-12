import { useState } from 'react';
import { useTeam } from '@/context/TeamContext';

export const useTeamSelectorState = () => {
  const { memberships, activeMembership, activeTeamId, isOwner, isSuperAdmin, selectTeam, reloadTeams } = useTeam();
  const [isOpen, setIsOpen] = useState<boolean>(false);
  const [isMembersModalOpen, setIsMembersModalOpen] = useState<boolean>(false);
  const [isCreateModalOpen, setIsCreateModalOpen] = useState<boolean>(false);

  const toggleDropdown = () => setIsOpen((prev) => !prev);
  const closeDropdown = () => setIsOpen(false);

  const handleSelectTeam = (teamId: string) => {
    selectTeam(teamId);
    setIsOpen(false);
  };

  const openMembersModal = () => {
    setIsOpen(false);
    setIsMembersModalOpen(true);
  };

  const closeMembersModal = () => setIsMembersModalOpen(false);

  const openCreateModal = () => {
    setIsOpen(false);
    setIsCreateModalOpen(true);
  };

  const closeCreateModal = () => setIsCreateModalOpen(false);

  return {
    memberships,
    activeMembership,
    activeTeamId,
    isOwner,
    isSuperAdmin,
    isOpen,
    isMembersModalOpen,
    isAddModalOpen: isMembersModalOpen,
    isCreateModalOpen,
    toggleDropdown,
    closeDropdown,
    handleSelectTeam,
    openMembersModal,
    openAddModal: openMembersModal,
    closeMembersModal,
    closeAddModal: closeMembersModal,
    openCreateModal,
    closeCreateModal,
    reloadTeams,
  };
};
