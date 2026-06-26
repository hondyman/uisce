import { createAI } from "ai/rsc";
import { submitUserMessage } from "./actions";

// Define the initial state of the AI. It can be any JSON object.
export const AI = createAI({
  actions: {
    submitUserMessage,
  },
  initialUIState: [],
  initialAIState: [],
});
