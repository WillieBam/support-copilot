import apiClient from '@/service/apiClient';
import type { Instruction, TeamInstructionResponse } from '@/types/instruction';

export const fetchTeamInstruction = async (teamId: string): Promise<TeamInstructionResponse> => {
    const response = await apiClient.get<TeamInstructionResponse>(`/api/teams/${teamId}/instruction`);
    return response.data;
};

export const saveTeamInstruction = async (teamId: string, instructionDetails: string): Promise<Instruction> => {
    const response = await apiClient.post<Instruction>(`/api/teams/${teamId}/instruction`, {
        instruction_details: instructionDetails,
    });
    return response.data;
};
