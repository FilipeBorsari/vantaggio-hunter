import { NextRequest, NextResponse } from "next/server";
import { callAI } from "@/lib/ai";

const SYSTEM = `Você é especialista em criação de templates para WhatsApp Business API com aprovação Meta.
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
}`;

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

  let text: string;
  try {
    text = await callAI({ system: SYSTEM, user: `Tipo de template: ${type}` });
  } catch (err) {
    console.error("AI error (all providers failed):", err);
    return NextResponse.json(
      { error: "Erro ao consultar a IA. Verifique a configuração das chaves de API." },
      { status: 502 },
    );
  }

  try {
    const raw = text.replace(/```json\n?/g, "").replace(/```\n?/g, "").trim();
    const parsed = JSON.parse(raw) as { template: string; variables: string[]; tips: string[] };
    return NextResponse.json(parsed);
  } catch {
    return NextResponse.json({ error: "Erro ao interpretar resposta da IA" }, { status: 500 });
  }
}
