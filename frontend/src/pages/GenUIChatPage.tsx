import React, { useState, useRef, useEffect } from 'react';
import { Send, Sparkles } from 'lucide-react';
import { ComparisonChart } from '../genui/components/ComparisonChart';
import { ComplianceDisclaimer } from '../genui/components/ComplianceDisclaimer';
import { ImpactAnalysisCard } from '../genui/components/ImpactAnalysisCard';

interface Message {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  display?: React.ReactNode;
  timestamp: Date;
}

export default function GenUIChatPage() {
  const [messages, setMessages] = useState<Message[]>([]);
  const [inputValue, setInputValue] = useState('');
  const [loading, setLoading] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  useEffect(() => {
    scrollToBottom();
  }, [messages]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!inputValue.trim() || loading) return;

    const userQuery = inputValue;
    setInputValue('');
    setLoading(true);

    const userMessage: Message = {
      id: `user-${Date.now()}`,
      role: 'user',
      content: userQuery,
      timestamp: new Date(),
    };

    setMessages(prev => [...prev, userMessage]);

    // Simulate AI response logic
    setTimeout(() => {
      let displayComponent: React.ReactNode | undefined;
      let textResponse = '';

      const queryLower = userQuery.toLowerCase();

      if (queryLower.includes('compare') || queryLower.includes('s&p 500') || queryLower.includes('tech')) {
        displayComponent = (
          <ComparisonChart 
            metric="Tech Exposure" 
            benchmark="S&P 500" 
            period="YTD" 
          />
        );
        textResponse = "Here is the comparison chart for your Tech Exposure vs the S&P 500 index YTD.";
      } else if (queryLower.includes('futures') || queryLower.includes('risk') || queryLower.includes('option')) {
        displayComponent = (
          <ComplianceDisclaimer 
            topic="Futures & Derivatives Trading" 
            severity="warning" 
          />
        );
        textResponse = "I have flagged this topic with the required compliance notice for Futures and regulated derivatives.";
      } else if (queryLower.includes('rate hikes') || queryLower.includes('real estate') || queryLower.includes('impact')) {
        displayComponent = (
          <ImpactAnalysisCard 
            headline="Fed signals further rate hikes amid persistent inflation" 
            affectedSector="Real Estate & Financials" 
            impactScore={82} 
          />
        );
        textResponse = "Here is the projected impact analysis of the latest interest rate hikes on interest-sensitive sectors.";
      } else {
        textResponse = `I received your query: "${userQuery}". You can try asking things like:
- "Compare Tech vs S&P 500"
- "What are the risks of Futures Trading?"
- "Analyze impact of rate hikes on Real Estate"`;
      }

      const assistantMessage: Message = {
        id: `assistant-${Date.now()}`,
        role: 'assistant',
        content: textResponse,
        display: displayComponent,
        timestamp: new Date(),
      };

      setMessages(prev => [...prev, assistantMessage]);
      setLoading(false);
    }, 1500);
  };

  return (
    <div className="flex flex-col h-[calc(100vh-80px)] bg-gray-50 dark:bg-gray-900">
      {/* Header */}
      <header className="bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 px-6 py-4 flex justify-between items-center">
        <div className="flex items-center gap-2">
          <div className="w-8 h-8 bg-blue-600 rounded-lg flex items-center justify-center text-white font-bold">
            W
          </div>
          <h1 className="text-xl font-bold text-gray-900 dark:text-gray-100">WealthStream GenUI Chat</h1>
        </div>
        <div className="flex items-center gap-1.5 text-xs text-gray-500 dark:text-gray-400">
          <Sparkles className="w-3.5 h-3.5 text-blue-500 animate-pulse" />
          Powered by Gemini Pro
        </div>
      </header>

      {/* Chat History */}
      <div className="flex-1 overflow-y-auto p-6 space-y-6">
        {messages.length === 0 ? (
          <div className="h-full flex flex-col items-center justify-center text-center text-gray-400 dark:text-gray-500 space-y-4">
            <div className="w-16 h-16 bg-gray-100 dark:bg-gray-800 rounded-full flex items-center justify-center">
              <span className="text-2xl">✨</span>
            </div>
            <div>
              <h2 className="text-lg font-medium text-gray-900 dark:text-gray-100">Generative UI Ready</h2>
              <p className="text-sm mt-1">Ask about your portfolio, compliance, or market events.</p>
            </div>
            
            <div className="grid grid-cols-1 md:grid-cols-3 gap-3 w-full max-w-2xl mt-8">
              <button 
                onClick={() => setInputValue("Compare my Tech exposure to S&P 500 YTD")}
                className="p-3 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-xl text-sm text-gray-700 dark:text-gray-300 hover:border-blue-500 hover:text-blue-600 transition-all text-left"
              >
                "Compare Tech vs S&P 500"
              </button>
              <button 
                onClick={() => setInputValue("What are the risks of Futures Trading?")}
                className="p-3 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-xl text-sm text-gray-700 dark:text-gray-300 hover:border-blue-500 hover:text-blue-600 transition-all text-left"
              >
                "Risks of Futures Trading"
              </button>
              <button 
                onClick={() => setInputValue("Analyze impact of rate hikes on Real Estate")}
                className="p-3 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-xl text-sm text-gray-700 dark:text-gray-300 hover:border-blue-500 hover:text-blue-600 transition-all text-left"
              >
                "Impact of Rate Hikes"
              </button>
            </div>
          </div>
        ) : (
          messages.map((message) => (
            <div key={message.id} className={`flex ${message.role === 'user' ? 'justify-end' : 'justify-start'}`}>
              <div className={`max-w-3xl ${message.role === 'user' ? 'ml-12' : 'mr-12 w-full'}`}>
                {message.role === 'user' ? (
                  <div className="bg-blue-600 text-white p-3.5 rounded-2xl rounded-tr-none shadow-sm text-sm">
                    {message.content}
                  </div>
                ) : (
                  <div className="space-y-2">
                    <div className="bg-white dark:bg-gray-800 text-gray-800 dark:text-gray-200 p-4 rounded-2xl rounded-tl-none shadow-sm border border-gray-100 dark:border-gray-700 text-sm whitespace-pre-wrap">
                      {message.content}
                    </div>
                    {message.display}
                  </div>
                )}
              </div>
            </div>
          ))
        )}
        
        {loading && (
          <div className="flex justify-start">
            <div className="bg-white dark:bg-gray-800 p-4 rounded-2xl rounded-tl-none shadow-sm border border-gray-100 dark:border-gray-700 flex items-center gap-1.5">
              <span className="w-2 h-2 bg-blue-600 rounded-full animate-bounce" style={{ animationDelay: '0ms' }} />
              <span className="w-2 h-2 bg-blue-600 rounded-full animate-bounce" style={{ animationDelay: '150ms' }} />
              <span className="w-2 h-2 bg-blue-600 rounded-full animate-bounce" style={{ animationDelay: '300ms' }} />
            </div>
          </div>
        )}
        <div ref={messagesEndRef} />
      </div>

      {/* Input Form */}
      <div className="p-4 bg-white dark:bg-gray-800 border-t border-gray-200 dark:border-gray-700">
        <form onSubmit={handleSubmit} className="max-w-4xl mx-auto relative flex items-center">
          <input
            type="text"
            value={inputValue}
            onChange={(e) => setInputValue(e.target.value)}
            placeholder="Ask a question or request a portfolio comparison..."
            className="w-full pl-4 pr-12 py-3.5 bg-gray-50 dark:bg-gray-900 border border-gray-200 dark:border-gray-700 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white dark:focus:bg-gray-950 transition-all text-sm text-gray-900 dark:text-gray-100"
            disabled={loading}
          />
          <button
            type="submit"
            disabled={!inputValue.trim() || loading}
            className="absolute right-3 p-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
          >
            <Send className="w-4 h-4" />
          </button>
        </form>
      </div>
    </div>
  );
}
