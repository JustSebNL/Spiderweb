import { task } from "@trigger.dev/sdk";

const DEFAULT_MODEL = "tencent/Youtu-LLM-2B";

type SpiderwebEvent = Record<string, unknown>;

function getBaseUrl(): string {
  return process.env.BRAIN_VLLM_BASE_URL ?? process.env.YOUTU_VLLM_BASE_URL ?? "http://127.0.0.1:8000/v1";
}

function getApiKey(): string {
  return process.env.BRAIN_VLLM_API_KEY ?? process.env.YOUTU_VLLM_API_KEY ?? "dummy";
}

async function chatCompletion(messages: Array<{ role: string; content: string }>) {
  const response = await fetch(`${getBaseUrl()}/chat/completions`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${getApiKey()}`,
    },
    body: JSON.stringify({
      model: DEFAULT_MODEL,
      temperature: 0.2,
      messages,
    }),
  });

  if (!response.ok) {
    const body = await response.text();
    throw new Error(`vLLM classify request failed: ${response.status} ${body}`);
  }

  const json = (await response.json()) as {
    choices?: Array<{ message?: { content?: string } }>;
  };

  return json.choices?.[0]?.message?.content ?? "";
}

export const classifyEvent = task({
  id: "classify-event",
  retry: {
    maxAttempts: 2,
  },
  run: async (payload: { event: SpiderwebEvent }) => {
    const content = await chatCompletion([
      {
        role: "system",
        content:
          "You are Spiderweb's cheap intake classifier. Return a compact JSON object with fields: priority, category, escalation_needed, one_line_summary.",
      },
      {
        role: "user",
        content: JSON.stringify(payload.event),
      },
    ]);

    return { content };
  },
});
