"use server";

import { createStreamableUI } from "ai/rsc";
import { google } from "@ai-sdk/google";
import { generateText } from "ai";
import { z } from "zod";
import { ComparisonChart } from "@/components/genui/ComparisonChart";
import { ComplianceDisclaimer } from "@/components/genui/ComplianceDisclaimer";
import { ImpactAnalysisCard } from "@/components/genui/ImpactAnalysisCard";

// Define the AI state and UI state types
export interface Message {
  role: "user" | "assistant";
  content: string;
  display?: React.ReactNode;
}

// The main server action to handle the chat
export async function submitUserMessage(input: string) {
  "use server";

  const ui = createStreamableUI(
    <div className="text-gray-500 animate-pulse">Analyzing intent...</div>
  );

  // We use an IIFE to allow the function to return the UI stream immediately
  // while the AI processing happens in the background
  (async () => {
    try {
      const response = await generateText({
        model: google("models/gemini-pro"),
        system: `You are an autonomous wealth management OS. 
        You help advisors by generating interactive UI components based on their intent.
        
        Available tools:
        - render_comparison_chart: For performance comparisons (e.g. "vs S&P 500")
        - render_compliance_disclaimer: When discussing regulated topics (Futures, Options, Muni Bonds)
        - render_impact_analysis: For news/market event impact on portfolios
        
        If no tool is needed, just reply with text.`,
        prompt: input,
        tools: {
          render_comparison_chart: {
            description: "Render a chart comparing portfolio/sector performance to a benchmark",
            parameters: z.object({
              metric: z.string().describe("The metric to compare, e.g., 'Tech Exposure'"),
              benchmark: z.string().describe("The benchmark, e.g., 'S&P 500'"),
              period: z.string().describe("Time period, e.g., 'YTD', '1Y'"),
            }),
            execute: async ({ metric, benchmark, period }) => {
              ui.update(<div className="text-blue-500">Fetching market data...</div>);
              
              // Simulate data fetch latency
              await new Promise((resolve) => setTimeout(resolve, 1000));

              // Return the interactive component
              ui.done(
                <ComparisonChart 
                  metric={metric} 
                  benchmark={benchmark} 
                  period={period} 
                />
              );
              return `Here is the comparison chart for ${metric} vs ${benchmark}.`;
            },
          },
          render_compliance_disclaimer: {
            description: "Render a compliance disclaimer for a specific topic",
            parameters: z.object({
              topic: z.string().describe("The regulated topic, e.g., 'Futures', 'Muni Bonds'"),
              severity: z.enum(["info", "warning", "critical"]).describe("Severity of the warning"),
            }),
            execute: async ({ topic, severity }) => {
              ui.done(
                <ComplianceDisclaimer 
                  topic={topic} 
                  severity={severity} 
                />
              );
              return `I've added the necessary disclaimer for ${topic}.`;
            },
          },
          render_impact_analysis: {
            description: "Render an impact analysis card for a news event",
            parameters: z.object({
              headline: z.string().describe("The news headline"),
              affected_sector: z.string().describe("The sector affected"),
              impact_score: z.number().describe("Impact score 0-100"),
            }),
            execute: async ({ headline, affected_sector, impact_score }) => {
              ui.done(
                <ImpactAnalysisCard 
                  headline={headline} 
                  affectedSector={affected_sector} 
                  impactScore={impact_score} 
                />
              );
              return `Here is the impact analysis for "${headline}".`;
            },
          },
        },
      });

      // If no tool was called, just stream the text
      if (response.text) {
        ui.done(<div className="whitespace-pre-wrap">{response.text}</div>);
      }
    } catch (error) {
      console.error("AI Error:", error);
      ui.done(<div className="text-red-500">Error generating response. Please try again.</div>);
    }
  })();

  return {
    id: Date.now(),
    display: ui.value,
  };
}
