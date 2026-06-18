import { NextRequest, NextResponse } from "next/server";
import { callAI } from "@/lib/ai";

const SYSTEM = `Você é especialista na Classificação Nacional de Atividades Econômicas (CNAE) brasileira.
Dada uma descrição de negócio em linguagem natural, retorne os CNAEs mais relevantes.
Responda APENAS com um JSON válido no formato: { "cnaes": [{ "code": "XXXXXXX", "description": "Descrição em português" }] }
Inclua entre 4 e 8 CNAEs ordenados por relevância.
Os códigos CNAE têm 7 dígitos no formato XXXXXXX (sem pontuação).`;

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

  let text: string;
  try {
    text = await callAI({ system: SYSTEM, user: `Negócio: ${description}` });
  } catch (err) {
    console.error("AI error (all providers failed):", err);
    return NextResponse.json(
      { error: "Erro ao consultar a IA. Verifique a configuração das chaves de API." },
      { status: 502 },
    );
  }

  try {
    const raw = text.replace(/```json\n?/g, "").replace(/```\n?/g, "").trim();
    const parsed = JSON.parse(raw) as { cnaes: { code: string; description: string }[] };
    return NextResponse.json(parsed);
  } catch {
    return NextResponse.json({ error: "Erro ao interpretar resposta da IA" }, { status: 500 });
  }
}
