import { NextRequest, NextResponse } from "next/server";
import Anthropic from "@anthropic-ai/sdk";

const client = new Anthropic();

export async function POST(req: NextRequest) {
  let type: string;
  try {
    const body = await req.json();
    type = body?.type?.trim();
  } catch {
    return NextResponse.json({ error: "Corpo da requisição inválido" }, { status: 400 });
  }

  if (!type) {
    return NextResponse.json({ error: "Tipo de template é obrigatório" }, { status: 400 });
  }

  const message = await client.messages.create({
    model: "claude-opus-4-8",
    max_tokens: 1024,
    system: `Você é especialista em criação de templates para WhatsApp Business API com aprovação Meta.
Crie templates de mensagem para prospecção de clientes que:
- Sejam aprovados pela Meta (sem linguagem promocional agressiva, sem spam)
- Usem linguagem profissional e respeitosa
- Incluam variáveis no formato {{1}}, {{2}} para personalização
- Tenham no máximo 1024 caracteres
- Sejam adequados para prospecção B2B
Responda APENAS com um JSON válido no formato:
{
  "template": "Texto do template com {{1}} para variáveis",
  "variables": ["Descrição da variável 1", "Descrição da variável 2"],
  "tips": ["Dica de aprovação 1", "Dica de aprovação 2"]
}`,
    messages: [
      {
        role: "user",
        content: `Tipo de template: ${type}`,
      },
    ],
  });

  const textBlock = message.content.find((b) => b.type === "text");
  if (!textBlock || textBlock.type !== "text") {
    return NextResponse.json({ error: "Erro ao processar resposta da IA" }, { status: 500 });
  }

  let parsed: { template: string; variables: string[]; tips: string[] };
  try {
    const raw = textBlock.text.replace(/```json\n?/g, "").replace(/```\n?/g, "").trim();
    parsed = JSON.parse(raw);
  } catch {
    return NextResponse.json({ error: "Erro ao interpretar resposta da IA" }, { status: 500 });
  }

  return NextResponse.json(parsed);
}
