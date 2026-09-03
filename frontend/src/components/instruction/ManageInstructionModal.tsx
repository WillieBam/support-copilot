import React, { useState, useEffect } from 'react';
import { FileText, History, Save, Loader2, X, CheckCircle2, AlertCircle, Clock, ChevronDown, ChevronUp } from 'lucide-react';
import type { Instruction, InstructionLog } from '@/types/instruction';
import { fetchTeamInstruction, saveTeamInstruction } from '@/service/instruction/instructionService';
import { useTeam } from '@/context/TeamContext';

interface ManageInstructionModalProps {
  teamId?: string | null;
  onClose: () => void;
}

export const ManageInstructionModal: React.FC<ManageInstructionModalProps> = ({
  teamId,
  onClose,
}) => {
  const { getUserDisplayName } = useTeam();
  const [instruction, setInstruction] = useState<Instruction | null>(null);
  const [logs, setLogs] = useState<InstructionLog[]>([]);
  const [details, setDetails] = useState('');
  const [isLoading, setIsLoading] = useState(true);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);
  const [showHistory, setShowHistory] = useState(false);

  const loadData = async () => {
    if (!teamId) return;
    setIsLoading(true);
    setError(null);
    try {
      const data = await fetchTeamInstruction(teamId);
      setInstruction(data.instruction);
      setLogs(data.logs || []);
      setDetails(data.instruction?.instruction_details || '');
    } catch (err: any) {
      setError(err?.response?.data?.error || 'Failed to load team instructions');
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, [teamId]);

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!teamId) return;
    if (!details.trim()) {
      setError('Instruction details cannot be empty.');
      return;
    }

    setIsSubmitting(true);
    setError(null);
    setSuccessMsg(null);

    try {
      await saveTeamInstruction(teamId, details.trim());
      setSuccessMsg('Team instruction updated successfully! Copilot will now use these guidelines.');
      await loadData();
      setTimeout(() => setSuccessMsg(null), 3500);
    } catch (err: any) {
      setError(err?.response?.data?.error || 'Failed to save team instruction');
    } finally {
      setIsSubmitting(false);
    }
  };

  const formatDate = (isoString?: string) => {
    if (!isoString) return '';
    try {
      return new Date(isoString).toLocaleString('en-MY', {
        timeZone: 'Asia/Kuala_Lumpur',
        month: 'short',
        day: 'numeric',
        year: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
        hour12: true,
      });
    } catch {
      return isoString;
    }
  };

  const latestLog = logs && logs.length > 0 ? logs[0] : null;

  return (
    <div className="fixed inset-0 z-100 flex items-center justify-center bg-background/80 backdrop-blur-sm p-4 overflow-y-auto">
      <div className="relative w-full max-w-2xl bg-card border border-border rounded-[20px] shadow-2xl p-6 transition-all my-8 max-h-[90vh] flex flex-col">
        <button
          onClick={onClose}
          className="absolute top-4 right-4 text-muted-foreground hover:text-foreground transition-colors cursor-pointer z-10"
        >
          <X className="w-5 h-5" />
        </button>

        <div className="mb-4 pr-8 border-b border-border pb-4">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 bg-emerald-500/10 border border-emerald-500/20 rounded-2xl flex items-center justify-center">
              <FileText className="w-5 h-5 text-emerald-500" />
            </div>
            <div>
              <h2 className="text-xl font-bold tracking-tight text-foreground">Manage Team Instructions</h2>
              <p className="text-xs text-muted-foreground mt-0.5">
                Customize Support Copilot behavior and rules tailored specifically for your team.
              </p>
            </div>
          </div>
        </div>

        {!teamId ? (
          <div className="py-12 text-center text-muted-foreground">
            <AlertCircle className="w-8 h-8 text-amber-500 mx-auto mb-2" />
            <p className="text-sm font-medium">No team selected</p>
            <p className="text-xs mt-1">Please select an active team in the header before configuring instructions.</p>
          </div>
        ) : isLoading ? (
          <div className="flex flex-col items-center justify-center py-16 text-muted-foreground">
            <Loader2 className="w-8 h-8 animate-spin text-emerald-500 mb-3" />
            <p className="text-sm">Loading team instructions...</p>
          </div>
        ) : (
          <div className="flex flex-col flex-1 overflow-y-auto pr-1 space-y-5">
            <form onSubmit={handleSave} className="space-y-4">
              <div>
                <div className="flex items-center justify-between mb-1.5">
                  <label className="text-sm font-semibold text-foreground">Custom LLM Guidelines & Prompt Rules</label>
                  {latestLog ? (
                    <span className="text-[11px] text-muted-foreground flex items-center gap-1">
                      <Clock className="w-3 h-3 text-emerald-500" /> Edited by: <strong className="text-foreground font-medium">{getUserDisplayName(latestLog.updated_by)}</strong> ({formatDate(instruction?.updated_at || instruction?.created_at)})
                    </span>
                  ) : instruction ? (
                    <span className="text-[11px] text-muted-foreground flex items-center gap-1">
                      <Clock className="w-3 h-3 text-emerald-500" /> Created by: <strong className="text-foreground font-medium">{getUserDisplayName(instruction.created_by)}</strong> ({formatDate(instruction.created_at)})
                    </span>
                  ) : null}
                </div>
                <textarea
                  value={details}
                  onChange={(e) => setDetails(e.target.value)}
                  rows={12}
                  placeholder="e.g. Always prioritize payment-gateway-service alerts. Provide concise container diagnostic steps and recommended kubectl remediation commands."
                  className="w-full min-h-[280px] bg-background border border-border rounded-xl p-4 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-emerald-500/50 resize-y leading-relaxed"
                  disabled={isSubmitting}
                />
              </div>

              {error && (
                <div className="p-3 bg-red-500/10 border border-red-500/20 text-red-500 rounded-xl text-xs flex items-center gap-2">
                  <AlertCircle className="w-4 h-4 shrink-0" /> {error}
                </div>
              )}

              {successMsg && (
                <div className="p-3 bg-emerald-500/10 border border-emerald-500/20 text-emerald-500 rounded-xl text-xs flex items-center gap-2">
                  <CheckCircle2 className="w-4 h-4 shrink-0" /> {successMsg}
                </div>
              )}

              <div className="flex items-center justify-between pt-2 border-t border-border">
                <button
                  type="button"
                  onClick={() => setShowHistory(!showHistory)}
                  className="flex items-center gap-1.5 text-xs text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
                >
                  <History className="w-3.5 h-3.5 text-emerald-500" />
                  Version History ({logs.length})
                  {showHistory ? <ChevronUp className="w-3.5 h-3.5" /> : <ChevronDown className="w-3.5 h-3.5" />}
                </button>

                <div className="flex gap-2">
                  <button
                    type="button"
                    onClick={onClose}
                    disabled={isSubmitting}
                    className="px-4 py-2 text-sm font-medium text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    disabled={isSubmitting || !details.trim()}
                    className="flex items-center gap-2 bg-emerald-500 hover:bg-emerald-600 text-white px-5 py-2.5 rounded-xl text-sm font-medium transition-colors disabled:opacity-50 cursor-pointer shadow-md"
                  >
                    {isSubmitting ? <Loader2 className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4" />}
                    Save Instruction
                  </button>
                </div>
              </div>
            </form>

            {/* Instruction Version History */}
            {showHistory && (
              <div className="pt-2 border-t border-border space-y-3">
                <h3 className="text-xs font-bold uppercase tracking-wider text-muted-foreground flex items-center gap-1.5">
                  <History className="w-3.5 h-3.5 text-emerald-500" /> Past Version Logs
                </h3>

                {logs.length === 0 ? (
                  <p className="text-xs text-muted-foreground italic bg-muted/20 p-3 rounded-xl">
                    No previous versions logged yet. Updates to instructions will automatically log older versions here.
                  </p>
                ) : (
                  <div className="space-y-2.5 max-h-48 overflow-y-auto pr-1">
                    {logs.map((log, idx) => {
                      const authorId = log.version === 1 || idx === logs.length - 1
                        ? instruction?.created_by
                        : logs[idx + 1]?.updated_by;
                      const versionTime = log.version === 1 || idx === logs.length - 1
                        ? instruction?.created_at
                        : log.updated_at;

                      return (
                        <div key={log.id} className="p-3 bg-muted/30 border border-border rounded-xl text-xs space-y-1">
                          <div className="flex items-center justify-between text-muted-foreground text-[11px]">
                            <span className="font-bold text-foreground">Version #{log.version}</span>
                            <span>{formatDate(versionTime)}</span>
                          </div>
                          <p className="font-mono text-muted-foreground bg-background/50 p-2 rounded-lg whitespace-pre-wrap text-[11px]">
                            {log.older_instruction}
                          </p>
                          <span className="text-[10px] text-muted-foreground/70 block">
                            {log.version === 1 ? 'Created by:' : 'Edited by:'} {getUserDisplayName(authorId)}
                          </span>
                        </div>
                      );
                    })}
                  </div>

                )}
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
};
