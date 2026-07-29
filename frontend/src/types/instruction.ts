export interface Instruction {
    id: string;
    created_by: string;
    team_id: string;
    instruction_details: string;
    created_at: string;
}

export interface InstructionLog {
    id: string;
    instruction_id: string;
    updated_by: string;
    older_instruction: string;
    version: number;
    updated_at: string;
}

export interface TeamInstructionResponse {
    instruction: Instruction | null;
    logs: InstructionLog[];
}
