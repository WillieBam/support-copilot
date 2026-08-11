import { useState, useEffect, useCallback } from 'react';
import { fetchTeamConversations, fetchConversationMessages } from '@/service/conversation/conversationService';
import type { Conversation, ConversationMessage } from '@/types/conversation';

export function useConversationState(activeTeamId: string | null) {
  const [recentConvs, setRecentConvs] = useState<Conversation[]>([]);
  const [allConvs, setAllConvs] = useState<Conversation[]>([]);
  const [selectedConvId, setSelectedConvId] = useState<string | null>(null);
  const [activeConvId, setActiveConvId] = useState<string | null>(null);
  const [chatKey, setChatKey] = useState<number>(0);
  const [selectedMessages, setSelectedMessages] = useState<ConversationMessage[]>([]);
  const [isReadOnly, setIsReadOnly] = useState<boolean>(false);
  const [isModalOpen, setIsModalOpen] = useState<boolean>(false);
  const [isLoadingRecent, setIsLoadingRecent] = useState<boolean>(false);
  const [isLoadingAll, setIsLoadingAll] = useState<boolean>(false);
  const [isLoadingMessages, setIsLoadingMessages] = useState<boolean>(false);

  // fetch top 15 recent conversations for the active team
  const loadRecent = useCallback(async () => {
    if (!activeTeamId) {
      setRecentConvs([]);
      return;
    }
    setIsLoadingRecent(true);
    try {
      const data = await fetchTeamConversations(activeTeamId, 15);
      setRecentConvs(data || []);
    } catch (err) {
      console.error('Failed to load recent conversations:', err);
    } finally {
      setIsLoadingRecent(false);
    }
  }, [activeTeamId]);

  // fetch all conversations for the active team
  const loadAll = useCallback(async () => {
    if (!activeTeamId) return;
    setIsLoadingAll(true);
    try {
      const data = await fetchTeamConversations(activeTeamId);
      setAllConvs(data || []);
    } catch (err) {
      console.error('Failed to load all conversations:', err);
    } finally {
      setIsLoadingAll(false);
    }
  }, [activeTeamId]);

  // open a specific past conversation in read-only mode
  const openConversation = useCallback(async (convId: string) => {
    setSelectedConvId(convId);
    setIsReadOnly(true);
    setIsModalOpen(false);
    setIsLoadingMessages(true);
    try {
      const msgs = await fetchConversationMessages(convId);
      setSelectedMessages(msgs || []);
    } catch (err) {
      console.error('Failed to load conversation messages:', err);
      setSelectedMessages([]);
    } finally {
      setIsLoadingMessages(false);
    }
  }, []);

  // return to live chat mode
  const closeReadOnly = useCallback(() => {
    setSelectedConvId(null);
    setSelectedMessages([]);
    setIsReadOnly(false);
  }, []);

  // open all conversations modal
  const openModal = useCallback(() => {
    setIsModalOpen(true);
    void loadAll();
  }, [loadAll]);

  // close modal
  const closeModal = useCallback(() => {
    setIsModalOpen(false);
  }, []);

  // start a fresh live conversation session
  const startNewChat = useCallback(() => {
    setActiveConvId(null);
    setChatKey((prev) => prev + 1);
    closeReadOnly();
  }, [closeReadOnly]);

  // handle callback when backend assigns a new conversation ID via SSE meta event
  const onConversationCreated = useCallback((newId: string) => {
    setActiveConvId(newId);
    void loadRecent();
  }, [loadRecent]);

  // handle callback when backend updates title for a conversation
  const onTitleGenerated = useCallback((convId: string, title: string) => {
    setRecentConvs((prev) =>
      prev.map((c) => (c.id === convId ? { ...c, title } : c))
    );
    setAllConvs((prev) =>
      prev.map((c) => (c.id === convId ? { ...c, title } : c))
    );
  }, []);

  // handle callback when chat completion stream ends cleanly
  const onFinish = useCallback(() => {
    void loadRecent();
  }, [loadRecent]);

  useEffect(() => {
    void loadRecent();
  }, [loadRecent]);

  // reset read-only when team changes
  useEffect(() => {
    closeReadOnly();
    setActiveConvId(null);
    setChatKey((prev) => prev + 1);
  }, [activeTeamId, closeReadOnly]);

  return {
    recentConvs,
    allConvs,
    selectedConvId,
    activeConvId,
    chatKey,
    selectedMessages,
    isReadOnly,
    isModalOpen,
    isLoadingRecent,
    isLoadingAll,
    isLoadingMessages,
    loadRecent,
    openConversation,
    closeReadOnly,
    openModal,
    closeModal,
    startNewChat,
    onConversationCreated,
    onTitleGenerated,
    onFinish,
    setActiveConvId,
  };
}
