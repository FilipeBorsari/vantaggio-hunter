import Anthropic from "@anthropic-ai/sdk";
import OpenAI from "openai";

export interface AIMessage {
  system: string;
  user: string;
  maxTokens?: number;
}

// Returns the text content from the AI, trying Anthropic first then OpenAI.
export async function callAI(msg: AIMessage): Promise<string> {
  const { system, user, maxTokens = 1024 } = msg;

  if (process.env.ANTHROPIC_API_KEY) {
    try {
      const client = new Anthropic();
      const response = await client.messages.create({
        model: "claude-opus-4-8",
        max_tokens: maxTokens,
        system,
        messages: [{ role: "user", content: user }],
      });
      const block = response.content.find((b) => b.type === "text");
      if (block && block.type === "text") return block.text;
    } catch (err) {
      console.warn("Anthropic failed, falling back to OpenAI:", (err as Error).message);
    }
  }

  if (process.env.OPENAI_API_KEY) {
    const client = new OpenAI();
    const response = await client.chat.completions.create({
      model: "gpt-4o-mini",
      max_tokens: maxTokens,
      messages: [
        { role: "system", content: system },
        { role: "user", content: user },
      ],
    });
    const text = response.choices[0]?.message?.content;
    if (text) return text;
  }

  throw new Error("Nenhuma chave de API configurada (ANTHROPIC_API_KEY ou OPENAI_API_KEY)");
}
