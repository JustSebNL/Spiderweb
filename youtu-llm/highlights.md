# Youtu-LLM Highlights

Source basis for this summary:
- `Youtu-LLM: Unlocking the Native Agentic Potential for Lightweight Large Language Models`
- Official `tencent/Youtu-LLM-2B` model card

This file is intentionally compressed for context-efficient reuse. It focuses on the model, the training recipe, the benchmark takeaways, and the practical implications.

## 1. What Youtu-LLM is

- Youtu-LLM is a lightweight open model built to show that strong agent behavior can be trained into a small model directly, instead of mostly relying on distillation or post-hoc scaffolding.
- The headline model is `Youtu-LLM 2B`, with `1.96B` parameters.
- Core claim: a small model can still gain native planning, reflection, tool-use, and long-horizon reasoning behavior if those capabilities are cultivated during pre-training and mid-training.
- The paper positions Youtu-LLM as an outlier on the parameter/performance curve for agentic tasks: unusually strong agentic performance for its size.

## 2. Why the paper matters

- The paper pushes a specific thesis: agentic competence should be trained early, not bolted on late.
- Instead of treating small-model reasoning as mostly an alignment or distillation problem, Youtu-LLM treats it as a curriculum and data construction problem.
- The work is notable because it combines:
  - compact architecture,
  - long context,
  - tokenizer design tuned for STEM and mixed-language text,
  - very large-scale pre-training,
  - explicit agentic trajectory data,
  - a staged transition from commonsense to STEM to agent behavior.

## 3. Model design

- Architecture type: autoregressive causal LM.
- Attention design: dense MLA, not MoE.
- Parameter count: `1.96B`.
- Layers: `32`.
- Hidden size: `2048`.
- Attention heads: `16` for Q/K/V in the MLA design.
- Context length: `131,072` tokens, often described as `128k`.
- Vocabulary size: `128,256`.

### Why dense MLA was chosen

- The authors explicitly avoid MoE for this setting because they argue MoE does not give compelling speed advantages for on-device scenarios and increases I/O complexity.
- MLA is used because it compresses KV cache and improves attention expressiveness relative to simpler grouped-query style designs.
- In the paper’s internal comparison, MLA outperformed GQA on both Chinese and English benchmarks for similarly sized 1B models trained from scratch.

## 4. Tokenizer design

- The tokenizer is a byte-level BPE tokenizer with a custom pre-tokenization strategy.
- It was designed to better support:
  - STEM text,
  - code,
  - mixed Chinese/English text,
  - cleaner semantic boundaries.
- Important tokenizer choices:
  - CJK characters and punctuation are segmented more strictly to avoid noisy cross-unit merges.
  - Numeric tokenization keeps only atomic digits `0-9`, avoiding multi-digit tokens.
  - The vocabulary starts from `o200k` but removes problematic Chinese tokens and invalid tokens under the new pre-tokenization scheme.
- The tokenizer was trained in stages instead of just fitting one mixed corpus directly.

### Tokenizer takeaway

- Their multi-stage tokenizer scored best among the tested options in the paper’s tokenizer comparison.
- The tokenizer is part of the model’s reasoning and STEM story, not just a preprocessing detail.

## 5. Training data and curriculum

## General pre-training corpus

- The authors collected over `10T` raw tokens, then filtered and deduplicated them.
- They retained `8.7T` raw tokens after filtering, then expanded and up-sampled parts of the data into a common pre-training pool of about `10.64T` tokens.
- English is primary; Chinese is secondary.
- The corpus heavily emphasizes:
  - web and encyclopedic knowledge,
  - STEM,
  - code.

### Domain emphasis

- `700B` tokens of Chinese and English STEM corpora.
- `1,400B` tokens of code source data.
- `500B` additional synthesized STEM/code corpora, including explanations, notebook-style content, summaries, paper interpretations, and code explanations.

### Data quality pipeline

- They built a classification and scoring system with:
  - `10` quality criteria,
  - `11` domain criteria,
  - `46` sub-domains.
- They used a semi-automated labeling process with manual verification.
- A smaller quality-selected subset reportedly outperformed more raw data at lower training cost, supporting the paper’s emphasis on quality over volume alone.

## Agentic trajectory corpus

- This is one of the most important parts of the paper.
- The model is trained on `200B` tokens of high-quality agentic trajectories.
- Breakdown reported in the paper:
  - `25B` Agentic-CoT trajectories,
  - `20B` math trajectories,
  - `70B` code trajectories,
  - `60B` deep research trajectories,
  - `25B` other trajectories such as tool use, function calling, and planning.

### Main thesis of the trajectory data

- The authors argue that native agent behavior comes from exposing the model to structured traces of planning, acting, checking, and correcting.
- They are not just tuning outputs to look agentic. They are trying to train the model to internalize agent patterns.

## 6. Agentic-CoT and the five-phase reasoning structure

- Youtu-LLM introduces an `Agentic-CoT` format that reorganizes long reasoning traces into cleaner structured phases.
- The five phases are:
  - `Analysis`
  - `Plan`
  - `Action`
  - `Reflection`
  - `Summary`

### Why this matters

- The paper argues ordinary long CoT often contains overthinking, repetition, and messy reasoning structure.
- Their rewrite process tries to preserve the useful logic while removing waste and making the reasoning trajectory more agent-shaped.
- This is central to the paper’s claim that planning and reflection can be learned as internal habits.

## 7. Math and code trajectory design

## Math trajectories

- Math is treated as a proxy for general agent cognition because it naturally expresses planning, execution, and reflection even without external tool feedback.
- The math data is built around an atomic ability hierarchy:
  - basic knowledge and computation,
  - complex reasoning and application,
  - mathematical metacognition.
- The agent loop for math is explicitly planning -> action -> feedback/reflection.

## Code trajectories

- Code trajectory data is a major part of the story: `70B` tokens.
- The paper highlights atomic code-agent skills such as:
  - exploration and localization in repositories,
  - context-aware code generation,
  - patch generation,
  - testing and verification,
  - environment comprehension,
  - self-reflection and correction.
- End-to-end code agent data is scaled through:
  - more tasks,
  - more repositories and contexts,
  - more action branches, including successful and failed paths.
- The code data explicitly uses multiple software-agent environments and scaffolds, but also prioritizes bash as the primitive tool layer.

## 8. Multi-stage training recipe

- The paper frames the whole recipe as `Commonsense -> STEM -> Agent`.
- There are four training stages:

### Stage 1: Commonsense pre-training

- Train from scratch.
- Sequence length: `8,192`.
- Consumes about `8.16T` tokens.
- Web pages and encyclopedic content dominate this stage.

### Stage 2: STEM- and coding-centric pre-training

- STEM and coding data is increased to about `60%`.
- This stage pushes structured reasoning and technical competence.

### Stage 3: General mid-training and long-context extension

- Context length is expanded from `8k -> 32k -> 128k`.
- Learning rate decays during this stage.
- Goal: stabilize STEM, coding, and long-context behavior.

### Stage 4: Agentic mid-training

- Roughly `60%` of the data becomes agentic trajectories.
- Learning rate decays much further.
- The paper says this stage works better after long-context training, likely because long-context ability helps the model track trajectory structure and state over long sequences.

### High-level takeaway

- The curriculum is the core idea.
- The model is not presented as a single clever architecture trick. It is presented as a carefully staged cognitive-development recipe.

## 9. Post-training

- Post-training includes:
  - supervised fine-tuning,
  - reinforcement learning.
- SFT data covers:
  - math,
  - code,
  - scientific reasoning,
  - agentic data,
  - general knowledge QA,
  - instruction following,
  - role-play,
  - creative writing,
  - multi-turn dialogue,
  - safety and alignment.

### Notable RL/training detail

- The paper emphasizes training stability and reports benefits from:
  - FP16 over BF16 in their setup,
  - consistent sampling to reduce drift between training policy and rollout policy.

## 10. Main benchmark takeaways

## Base model

- The base model is unusually strong for its size.
- It beats similarly sized baselines on many commonsense, STEM, and coding tasks.
- It is competitive with, and sometimes close to, larger models such as `Qwen3 4B Base`.

### Base model notable scores

- `MMLU-Pro`: `48.4`
- `GSM8K`: `77.6`
- `MGSM-Zh`: `68.9`
- `MATH`: `44.4`
- `HLE-MC`: `17.4`
- `HumanEval`: `64.6`
- `HumanEval+`: `57.3`
- `MBPP+`: `81.8`
- `LiveCodeBench v6`: `9.7`
- `NIAH`: `98.8`

### Base model agent benchmark result

- On `APTBench`, Youtu-LLM 2B Base is strong across code, deep research, math, and tool categories.
- Reported scores:
  - Code: `37.9`
  - Deep Research: `38.6`
  - Math: `68.0`
  - Tool: `64.2`

## Instruct model

- The instruct model is the stronger practical story.
- It beats or closely tracks larger models on several reasoning and coding tasks.
- It is especially impressive on coding and agent benchmarks given the parameter count.

### Instruct model notable scores

- `IFEval`: `81.2`
- `DROP`: `86.7`
- `MATH-500`: `93.7`
- `AIME 24`: `65.4`
- `AIME 25`: `49.8`
- `GPQA-Diamond`: `48.0`
- `HumanEval`: `95.9`
- `HumanEval+`: `89.0`
- `MBPP`: `85.0`
- `LiveCodeBench v6`: `43.7`

### Agent benchmark highlights

- `GAIA`: `33.9`
- `xbench`: `19.5`
- `SWE-Bench-Verified`: `17.7`
- `EnConda-Bench`: `21.5`
- `BFCL V3`: `58.0`
- `tau^2-Bench`: `15.0`

### Most important practical conclusion

- The paper’s strongest evidence is not just general benchmark quality.
- The strongest evidence is that a sub-2B model can become a meaningfully capable agent model on:
  - software issue resolution,
  - deep research,
  - tool invocation,
  - multi-step planning tasks.

## 11. Agentic mid-training result

- This is one of the paper’s clearest results.
- The authors report a roughly logarithmic gain curve as agentic training tokens increase.
- A large fraction of the gains appears early, within the first `34B` tokens of agentic mid-training.
- Across the full training budget, agentic mid-training improves average agent performance by more than `6%`.

### With vs without agentic mid-training

- `GAIA`: `31.1 -> 33.9`
- `xbench`: `18.0 -> 19.5`
- `SWE-Bench-Verified`: `12.4 -> 17.7`
- `EnConda-Bench`: `19.8 -> 21.5`
- `BFCL V3`: `59.3 -> 58.0` only notable regression
- `tau^2-Bench`: `12.5 -> 15.0`

### Interpretation

- Agentic mid-training is the strongest causal claim in the paper.
- It appears to help most on long-horizon or execution-heavy tasks, especially software engineering.
- The large jump on SWE-Bench-Verified is particularly important because it suggests the model is not just becoming better at looking agentic in text; it is becoming more useful in actual task completion.

## 12. Practical usage notes from the official release

- Released as both `Base` and `Instruct`.
- Supports a dedicated reasoning mode via `enable_thinking=True`.
- The model emits explicit `<think>...</think>` style reasoning traces in that mode.
- Recommended generation settings for reasoning mode:
  - `do_sample=True`
  - `temperature=1.0`
  - `top_p=0.95`
  - `top_k=20`
  - `repetition_penalty=1.05`
- Direct answer mode is also supported with `enable_thinking=False`.

### Deployment notes

- Hugging Face support is provided.
- The model card mentions support for:
  - Transformers,
  - vLLM `0.10.2`,
  - llama.cpp deployment paths.

## 13. Limitations called out by the paper

- There is still a gap versus larger proprietary frontier models.
- Long reasoning traces increase inference latency.
- The current system is text-only.
- Future direction proposed by the authors:
  - move toward stronger world-model style grounding,
  - improve efficiency further,
  - extend to multimodal or omni-modal settings.

## 14. Bottom-line summary

- Youtu-LLM is not just a small general LLM with decent benchmarks.
- Its main contribution is a training recipe for making a small model natively more agentic.
- The most important ingredients are:
  - dense MLA architecture,
  - a STEM-oriented tokenizer,
  - a large high-quality corpus,
  - a staged Commonsense-STEM-Agent curriculum,
  - large-scale agentic trajectory pre-training,
  - explicit planning/action/reflection data structures.
- The most important result is that these choices produce a `1.96B` model that is unusually strong on code, deep research, and other agent-style tasks relative to its size.
- If you care about low-cost deployment, long context, and agentic behavior in a compact model, this paper is worth paying attention to.

## 15. Why it could be a great fit for Spiderweb

Spiderweb is trying to be an ultra-lightweight, agent-shaped assistant that can operate in constrained environments, track state across long workflows, and do useful real work such as coding, planning, routing, tool use, and intake. That makes Youtu-LLM interesting for structural reasons, not just benchmark reasons.

### Strong reasons to consider it

- It is small enough to be operationally realistic. At `1.96B`, it is far closer to Spiderweb's footprint goals than mainstream 7B+ agent models.
- It was explicitly trained for agent behavior. Spiderweb is not just a chatbot; it needs planning, action sequencing, reflection, and tool-use patterns. Youtu-LLM was trained on exactly that kind of trajectory data.
- It is unusually strong on software and agent benchmarks for its size. That matters because Spiderweb includes coding, automation, transfer, and intake-style workflows.
- It supports `128k` context. That is useful for long issue threads, repo summaries, transfer sheets, onboarding state, and multi-step task continuity.
- It appears strong in code-oriented settings. Spiderweb already has a strong engineering/operator shape, so a model that is better at repository exploration, patching, and verification is a better fit than a general chat-optimized small model.
- It aligns with the project's low-cost hardware story better than larger reasoning models do.

### Where it maps directly to Spiderweb features

- `agent` and `skills`: Youtu-LLM's training emphasis on plan/action/reflection is a good fit for command-driven agent loops.
- `transfer` and `openclaw`: long-context plus agentic state tracking makes it a plausible intake or triage model before escalation to a larger system.
- `onboard`: the model's structured reasoning style could help with guided setup, config generation, and environment diagnosis.
- `cron`, `gateway`, and general automation: the paper's tool-use and planning emphasis matches event-driven or scheduled task execution better than generic instruction-tuned small models.

### Best strategic use inside Spiderweb

- Use it as the default low-cost local reasoning and intake model.
- Let it handle first-pass planning, tool routing, repo exploration, draft patches, summarization, and state tracking.
- Escalate only harder cases to a larger external model. This matches Spiderweb's "small fast local model first, expensive model second" philosophy.

### Real caveats

- `1.96B` is small for an LLM, but not tiny for very weak devices. The released weights are still multiple GB before quantization, so the deployment target matters.
- The paper proves relative strength, not guaranteed real-time performance on Spiderweb's lowest-end hardware.
- The license is custom (`youtu-llm`), so commercial and redistribution constraints need review before deeper adoption.
- Text-only is fine for current Spiderweb CLI flows, but it does not help future multimodal ambitions.
- Strong benchmark performance does not remove the need for task-specific evals on Spiderweb commands and agent loops.

### Bottom-line recommendation for this project

Youtu-LLM looks like a strong candidate if Spiderweb wants a compact local model with better-than-expected agent behavior, especially for coding, planning, long-context intake, and tool-oriented workflows. The most credible role is not "one model for everything"; it is "the default small local worker that handles most operational tasks before handing off edge cases." That is where the paper's evidence lines up best with this project.
