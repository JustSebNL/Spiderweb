FROM vllm/vllm-openai:v0.10.2

# Command to run vLLM server with Youtu-LLM-2B
# The --model argument specifies the Hugging Face model ID
CMD ["python", "-m", "vllm.entrypoints.openai.api_server", "--model", "tencent/Youtu-LLM-2B", "--host", "0.0.0.0", "--port", "8000"]
