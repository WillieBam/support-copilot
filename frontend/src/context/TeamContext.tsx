import { createContext, useContext, type ReactNode } from 'react';
import { useTeamState } from '@/hooks/useTeamState';

type TeamContextType = ReturnType<typeof useTeamState>;

const TeamContext = createContext<TeamContextType | null>(null);

export const TeamProvider = ({
  children,
  isSignedIn,
  userEmail,
}: {
  children: ReactNode;
  isSignedIn: boolean;
  userEmail?: string;
}) => {
  const teamState = useTeamState(isSignedIn, userEmail);
  return <TeamContext.Provider value={teamState}>{children}</TeamContext.Provider>;
};

export const useTeam = () => {
  const context = useContext(TeamContext);
  if (!context) {
    throw new Error('useTeam must be used within TeamProvider');
  }
  return context;
};