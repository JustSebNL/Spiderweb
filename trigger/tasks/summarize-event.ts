import { task } from "@trigger.dev/sdk";

const DEFAULT_MODEL = "tencent/Youtu-LLM-2B";

function getBaseUrl(): string {
  return process.env.BRAIN_VLLM_BASE_URL ?? process.env.YOUTU_VLLM_BASE_URL ?? "http://127.0.0.1:8000/v1";
}

function getApiKey(): string {
  return process.env.BRAIN_VLLM_API_KEY ?? process.env.YOUTU_VLLM_API_KEY ?? "dummy";
}

async function summarize(input: string) {
  const response = await fetch(`${getBaseUrl()}/chat/completions`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${getApiKey()}`,
    },
    body: JSON.stringify({
      model: DEFAULT_MODEL,
      temperature: 0.1,
      messages: [
        {
          role: "system",
          content: "Summarize the input into a short operational summary for Spiderweb intake. Keep it under 4 bullet-equivalent sentences.",
        },
        {
          role: "user",
          content: input,
        },
      ],
    }),
  });

  if (!response.ok) {
    const body = await response.text();
    throw new Error(`vLLM summarize request failed: ${response.status} ${body}`);
  }

  const json = (await response.json()) as {
    choices?: Array<{ message?: { content?: string } }>;
  };

  return json.choices?.[0]?.message?.content ?? "";
}

export const summarizeEvent = task({
  id: "summarize-event",
  retry: {
    maxAttempts: 2,
  },
  run: async (payload: { input: string }) => {
    const content = await summarize(payload.input);
    return { content };
  },
});
