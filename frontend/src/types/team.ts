export type TeamRole = 'owner' | 'member' | 'admin';

export interface TeamMember {
    id: string;
    team_id: string;
    user_id: string;
    role: TeamRole;
    user?: {
        id: string;
        email: string;
        display_name?: string;
    };
}

export interface Team {
    id: string;
    team_name: string;
    created_at: string;
    members?: TeamMember[];
}

export interface UserMembership {
  id: string;
  team_id: string;
  user_id: string;
  role: TeamRole;
  team: Team;
}

export interface UserWithTeams {
  id: string;
  email: string;
  firebase_uid: string;
  scope: string;
  memberships: UserMembership[];
}

export interface AddMemberPayload {
  user_id: string;
}
