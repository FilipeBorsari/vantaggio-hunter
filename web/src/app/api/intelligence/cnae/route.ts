import { NextRequest, NextResponse } from "next/server";
import Anthropic from "@anthropic-ai/sdk";

const client = new Anthropic();

export async function POST(req: NextRequest) {
  let description: string;
  try {
    const body = await req.json();
    description = body?.description?.trim();
  } catch {
    return NextResponse.json({ error: "Corpo da requisição inválido" }, { status: 400 });
  }

  if (!description) {
    return NextResponse.json({ error: "Descrição do negócio é obrigatória" }, { status: 400 });
  }

  const message = await client.messages.create({
    model: "claude-opus-4-8",
    max_tokens: 1024,
    system: `Você é especialista na Classificação Nacional de Atividades Econômicas (CNAE) brasileira.
Dada uma descrição de negócio em linguagem natural, retorne os CNAEs mais relevantes.
Responda APENAS com um JSON válido no formato: { "cnaes": [{ "code": "XXXXXXX", "description": "Descrição em português" }] }
Inclua entre 4 e 8 CNAEs ordenados por relevância.
Os códigos CNAE têm 7 dígitos no formato XXXXXXX (sem pontuação).`,
    messages: [
      {
        role: "user",
        content: `Negócio: ${description}`,
      },
    ],
  });

  const textBlock = message.content.find((b) => b.type === "text");
  if (!textBlock || textBlock.type !== "text") {
    return NextResponse.json({ error: "Erro ao processar resposta da IA" }, { status: 500 });
  }

  let parsed: { cnaes: { code: string; description: string }[] };
  try {
    const raw = textBlock.text.replace(/```json\n?/g, "").replace(/```\n?/g, "").trim();
    parsed = JSON.parse(raw);
  } catch {
    return NextResponse.json({ error: "Erro ao interpretar resposta da IA" }, { status: 500 });
  }

  return NextResponse.json(parsed);
}
