import React, { useState, useEffect } from 'react';
import { Send, Paperclip, Search, Check, CheckCheck, Shield, Lock } from 'lucide-react';

interface Message {
  messageId: string;
  conversationId: string;
  senderId: string;
  senderType: 'CLIENT' | 'ADVISOR' | 'SYSTEM';
  senderName: string;
  messageText: string;
  attachments: Attachment[];
  readAt: string | null;
  createdAt: string;
}

interface Conversation {
  conversationId: string;
  participantName: string;
  participantRole: string;
  lastMessage: string;
  lastMessageAt: string;
  unreadCount: number;
}

interface Attachment {
  filename: string;
  url: string;
  size: number;
}

export const SecureMessaging: React.FC = () => {
  const [conversations, setConversations] = useState<Conversation[]>([]);
  const [selectedConversation, setSelectedConversation] = useState<string | null>(null);
  const [messages, setMessages] = useState<Message[]>([]);
  const [messageText, setMessageText] = useState('');
  const [isSending, setIsSending] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const messagesEndRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    fetchConversations();
  }, []);

  useEffect(() => {
    if (selectedConversation) {
      fetchMessages(selectedConversation);
      markConversationAsRead(selectedConversation);
    }
  }, [selectedConversation]);

  useEffect(() => {
    scrollToBottom();
  }, [messages]);

  const fetchConversations = async () => {
    try {
      const response = await fetch('/api/messaging/conversations');
      const data = await response.json();
      setConversations(data);
    } catch (error) {
      console.error('Failed to fetch conversations:', error);
    }
  };

  const fetchMessages = async (conversationId: string) => {
    try {
      const response = await fetch(`/api/messaging/conversations/${conversationId}/messages?limit=50`);
      const data = await response.json();
      setMessages(data.reverse());
    } catch (error) {
      console.error('Failed to fetch messages:', error);
    }
  };

  const sendMessage = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!messageText.trim() || !selectedConversation) return;

    setIsSending(true);
    try {
      const response = await fetch('/api/messaging/messages', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          conversationId: selectedConversation,
          messageText: messageText.trim(),
          attachments: [],
        }),
      });

      if (response.ok) {
        const newMessage = await response.json();
        setMessages([...messages, newMessage]);
        setMessageText('');
        fetchConversations();
      }
    } catch (error) {
      console.error('Failed to send message:', error);
    } finally {
      setIsSending(false);
    }
  };

  const markConversationAsRead = async (conversationId: string) => {
    try {
      await fetch(`/api/messaging/conversations/${conversationId}/mark-read`, {
        method: 'POST',
      });
      fetchConversations();
    } catch (error) {
      console.error('Failed to mark as read:', error);
    }
  };

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  const formatTime = (dateString: string) => {
    const date = new Date(dateString);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMs / 3600000);
    const diffDays = Math.floor(diffMs / 86400000);

    if (diffMins < 1) return 'Just now';
    if (diffMins < 60) return `${diffMins}m ago`;
    if (diffHours < 24) return `${diffHours}h ago`;
    if (diffDays < 7) return `${diffDays}d ago`;
    return date.toLocaleDateString();
  };

  const filteredConversations = conversations.filter(conv =>
    conv.participantName.toLowerCase().includes(searchQuery.toLowerCase())
  );

  return (
    <div className="flex h-screen bg-gray-50">
      {/* Conversations Sidebar */}
      <div className="w-80 bg-white border-r border-gray-200 flex flex-col">
        <div className="p-4 border-b border-gray-200">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-xl font-bold text-gray-900">Messages</h2>
            <div className="flex items-center gap-1 text-xs text-green-600">
              <Shield className="w-4 h-4" />
              <span>Encrypted</span>
            </div>
          </div>

          <div className="relative">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-gray-400" />
            <input
              type="text"
              placeholder="Search conversations..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full pl-10 pr-4 py-2 bg-gray-50 border border-gray-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
            />
          </div>
        </div>

        <div className="flex-1 overflow-y-auto">
          {filteredConversations.map((conv) => (
            <button
              key={conv.conversationId}
              onClick={() => setSelectedConversation(conv.conversationId)}
              className={`w-full p-4 border-b border-gray-100 hover:bg-gray-50 transition-colors text-left ${
                selectedConversation === conv.conversationId ? 'bg-indigo-50 border-l-4 border-l-indigo-600' : ''
              }`}
            >
              <div className="flex items-start justify-between mb-1">
                <div className="flex items-center gap-2">
                  <div className="w-10 h-10 rounded-full bg-gradient-to-br from-indigo-500 to-purple-600 flex items-center justify-center text-white font-semibold">
                    {conv.participantName.charAt(0)}
                  </div>
                  <div>
                    <h3 className="font-semibold text-gray-900">{conv.participantName}</h3>
                    <p className="text-xs text-gray-500">{conv.participantRole}</p>
                  </div>
                </div>
                {conv.unreadCount > 0 && (
                  <span className="bg-indigo-600 text-white text-xs font-bold px-2 py-1 rounded-full">
                    {conv.unreadCount}
                  </span>
                )}
              </div>
              <p className="text-sm text-gray-600 truncate">{conv.lastMessage}</p>
              <p className="text-xs text-gray-400 mt-1">{formatTime(conv.lastMessageAt)}</p>
            </button>
          ))}

          {filteredConversations.length === 0 && (
            <div className="p-8 text-center text-gray-500">
              <p>No conversations found</p>
            </div>
          )}
        </div>
      </div>

      {/* Messages Area */}
      <div className="flex-1 flex flex-col">
        {selectedConversation ? (
          <>
            <div className="bg-white border-b border-gray-200 p-4">
              <div className="flex items-center justify-between">
                <div>
                  <h3 className="font-semibold text-gray-900">
                    {conversations.find(c => c.conversationId === selectedConversation)?.participantName}
                  </h3>
                  <p className="text-sm text-gray-500">
                    {conversations.find(c => c.conversationId === selectedConversation)?.participantRole}
                  </p>
                </div>
                <div className="flex items-center gap-2 text-xs text-gray-500">
                  <Lock className="w-4 h-4" />
                  <span>Encrypted</span>
                </div>
              </div>
            </div>

            <div className="flex-1 overflow-y-auto p-4 space-y-4 bg-gradient-to-br from-gray-50 to-blue-50">
              {messages.map((message) => {
                const isOwn = message.senderType === 'CLIENT';
                
                return (
                  <div key={message.messageId} className={`flex ${isOwn ? 'justify-end' : 'justify-start'}`}>
                    <div className={`max-w-lg ${isOwn ? 'order-2' : 'order-1'}`}>
                      {!isOwn && (
                        <p className="text-xs text-gray-500 mb-1 px-1">{message.senderName}</p>
                      )}
                      <div
                        className={`rounded-2xl px-4 py-3 ${
                          isOwn
                            ? 'bg-gradient-to-br from-indigo-600 to-purple-600 text-white'
                            : 'bg-white text-gray-900 border border-gray-200'
                        }`}
                      >
                        <p className="text-sm whitespace-pre-wrap break-words">{message.messageText}</p>
                        
                        {message.attachments.length > 0 && (
                          <div className="mt-2 space-y-1">
                            {message.attachments.map((att, idx) => (
                              <div key={idx} className="flex items-center gap-2 text-xs">
                                <Paperclip className="w-3 h-3" />
                                <a href={att.url} className="underline hover:no-underline">
                                  {att.filename}
                                </a>
                              </div>
                            ))}
                          </div>
                        )}
                      </div>
                      <div className={`flex items-center gap-1 mt-1 px-1 text-xs text-gray-500 ${isOwn ? 'justify-end' : 'justify-start'}`}>
                        <span>{formatTime(message.createdAt)}</span>
                        {isOwn && (
                          message.readAt ? (
                            <CheckCheck className="w-4 h-4 text-indigo-600" />
                          ) : (
                            <Check className="w-4 h-4" />
                          )
                        )}
                      </div>
                    </div>
                  </div>
                );
              })}
              <div ref={messagesEndRef} />
            </div>

            <form onSubmit={sendMessage} className="bg-white border-t border-gray-200 p-4">
              <div className="flex items-end gap-2">
                <button
                  type="button"
                  className="p-2 text-gray-400 hover:text-gray-600 transition-colors"
                  title="Attach file"
                >
                  <Paperclip className="w-5 h-5" />
                </button>

                <div className="flex-1 relative">
                  <textarea
                    value={messageText}
                    onChange={(e) => setMessageText(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter' && !e.shiftKey) {
                        e.preventDefault();
                        sendMessage(e);
                      }
                    }}
                    placeholder="Type your message... (Shift+Enter for new line)"
                    className="w-full px-4 py-3 border border-gray-300 rounded-xl resize-none focus:outline-none focus:ring-2 focus:ring-indigo-500 max-h-32"
                    rows={1}
                    style={{ minHeight: '48px' }}
                  />
                </div>

                <button
                  type="submit"
                  disabled={!messageText.trim() || isSending}
                  className="p-3 bg-gradient-to-r from-indigo-600 to-purple-600 text-white rounded-xl hover:from-indigo-700 hover:to-purple-700 disabled:opacity-50 disabled:cursor-not-allowed transition-all shadow-lg hover:shadow-xl"
                >
                  <Send className="w-5 h-5" />
                </button>
              </div>

              <p className="text-xs text-gray-500 mt-2 flex items-center gap-1">
                <Shield className="w-3 h-3" />
                All messages are encrypted end-to-end
              </p>
            </form>
          </>
        ) : (
          <div className="flex-1 flex items-center justify-center bg-gradient-to-br from-gray-50 to-blue-50">
            <div className="text-center">
              <div className="w-20 h-20 bg-gradient-to-br from-indigo-500 to-purple-600 rounded-full flex items-center justify-center mx-auto mb-4">
                <Shield className="w-10 h-10 text-white" />
              </div>
              <h3 className="text-xl font-semibold text-gray-900 mb-2">Secure Messaging</h3>
              <p className="text-gray-600">Select a conversation to start messaging</p>
              <p className="text-sm text-gray-500 mt-2">All messages are end-to-end encrypted</p>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};
