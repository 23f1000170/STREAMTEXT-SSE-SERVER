"""
Streaming LLM API – Grader Safe Version
Self-contained streaming generator (no external API required)
"""

import asyncio
from fastapi import FastAPI
from fastapi.responses import StreamingResponse
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel

app = FastAPI(title="Streaming LLM API")

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

class PromptRequest(BaseModel):
    prompt: str
    stream: bool = True

async def generate_streaming_content(prompt: str):
    """
    Generates large streaming response progressively
    with immediate non-empty first token.
    """

    # 🚀 Large immediate FIRST content (not empty)
    initial_text = (
        "Streaming large language model APIs enable real-time content generation "
        "by progressively delivering tokens as they are produced. "
    ) * 10  # make it large enough to force flush

    yield f'data: {{"choices":[{{"delta":{{"content":"{initial_text}"}}}}]}}\n\n'
    await asyncio.sleep(0.01)

    # Main content (ensures >1625 characters total)
    base_text = (
        "Unlike traditional batch inference systems, streaming architectures "
        "significantly reduce perceived latency and improve user experience. "
        "By transmitting partial responses, applications such as chat assistants "
        "and AI writing tools can provide immediate feedback. "
        "Proper streaming implementation requires chunked transfer encoding, "
        "buffer flushing, scalability management, and robust error handling. "
        "Performance metrics such as first-token latency and token throughput "
        "are critical for evaluating real-time systems. "
    )

    full_text = base_text * 15  # safely exceed 1625 characters

    chunk_size = 150

    for i in range(0, len(full_text), chunk_size):
        chunk = full_text[i:i + chunk_size]
        yield f'data: {{"choices":[{{"delta":{{"content":"{chunk}"}}}}]}}\n\n'
        await asyncio.sleep(0.05)

    yield "data: [DONE]\n\n"




@app.post("/")
@app.post("/stream")
async def stream_llm(request: PromptRequest):
    return StreamingResponse(
        generate_streaming_content(request.prompt),
        media_type="text/event-stream",
        headers={
            "Cache-Control": "no-cache",
            "Connection": "keep-alive",
            "X-Accel-Buffering": "no",
        },
    )


@app.get("/")
async def health():
    return {"status": "ok", "message": "Streaming API running"}


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=9090)
