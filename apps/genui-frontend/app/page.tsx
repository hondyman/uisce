"use client";

import { useState } from "react";
import { useActions, useUIState } from "ai/rsc";
import type { Message } from "./actions";
import { Send } from "lucide-react";

export default function Home() {
  const [inputValue, setInputValue] = useState("");
  const [messages, setMessages] = useUIState();
  const { submitUserMessage } = useActions();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!inputValue.trim()) return;

    // Add user message to UI state
    setMessages((currentMessages: Message[]) => [
      ...currentMessages,
      {
        id: Date.now(),
        role: "user",
        display: <div className="bg-gray-100 p-3 rounded-lg text-gray-800">{inputValue}</div>,
      },
    ]);

    const value = inputValue;
    setInputValue("");

    // Call server action
    const responseMessage = await submitUserMessage(value);
    
    // Add assistant response to UI state
    setMessages((currentMessages: Message[]) => [
      ...currentMessages,
      responseMessage,
    ]);
  };

  return (
    <div className="flex flex-col h-screen bg-gray-50">
      {/* Header */}
      <header className="bg-white border-b border-gray-200 px-6 py-4 flex justify-between items-center sticky top-0 z-10">
        <div className="flex items-center gap-2">
          <div className="w-8 h-8 bg-blue-600 rounded-lg flex items-center justify-center text-white font-bold">
            W
          </div>
          <h1 className="text-xl font-bold text-gray-900">WealthStream OS</h1>
        </div>
        <div className="text-sm text-gray-500">
          Powered by Gemini Pro & Vercel AI SDK
        </div>
      </header>

      {/* Chat Area */}
      <div className="flex-1 overflow-y-auto p-6 space-y-6">
        {messages.length === 0 ? (
          <div className="h-full flex flex-col items-center justify-center text-center text-gray-400 space-y-4">
            <div className="w-16 h-16 bg-gray-100 rounded-full flex items-center justify-center">
              <span className="text-2xl">✨</span>
            </div>
            <div>
              <h2 className="text-lg font-medium text-gray-900">Generative UI Ready</h2>
              <p>Ask about your portfolio, compliance, or market impact.</p>
            </div>
            <div className="grid grid-cols-1 md:grid-cols-3 gap-2 w-full max-w-2xl mt-8">
              <button 
                onClick={() => setInputValue("Compare my Tech exposure to S&P 500 YTD")}
                className="p-3 bg-white border border-gray-200 rounded-lg text-sm hover:border-blue-500 hover:text-blue-600 transition-colors"
              >
                "Compare Tech vs S&P 500"
              </button>
              <button 
                onClick={() => setInputValue("What are the risks of Futures Trading?")}
                className="p-3 bg-white border border-gray-200 rounded-lg text-sm hover:border-blue-500 hover:text-blue-600 transition-colors"
              >
                "Risks of Futures Trading"
              </button>
              <button 
                onClick={() => setInputValue("Analyze impact of rate hikes on Real Estate")}
                className="p-3 bg-white border border-gray-200 rounded-lg text-sm hover:border-blue-500 hover:text-blue-600 transition-colors"
              >
                "Impact of Rate Hikes"
              </button>
            </div>
          </div>
        ) : (
          messages.map((message: any) => (
            <div key={message.id} className={`flex ${message.role === 'user' ? 'justify-end' : 'justify-start'}`}>
              <div className={`max-w-3xl ${message.role === 'user' ? 'ml-12' : 'mr-12 w-full'}`}>
                {message.display}
              </div>
            </div>
          ))
        )}
      </div>

      {/* Input Area */}
      <div className="p-4 bg-white border-t border-gray-200">
        <form onSubmit={handleSubmit} className="max-w-4xl mx-auto relative">
          <input
            type="text"
            value={inputValue}
            onChange={(e) => setInputValue(e.target.value)}
            placeholder="Ask anything..."
            className="w-full pl-4 pr-12 py-4 bg-gray-50 border border-gray-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white transition-all shadow-sm"
          />
          <button
            type="submit"
            aria-label="Send message"
            disabled={!inputValue.trim()}
            className="absolute right-3 top-1/2 -translate-y-1/2 p-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            <Send className="w-4 h-4" />
          </button>
        </form>
      </div>
    </div>
  );
}
