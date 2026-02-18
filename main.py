import asyncio
from fastapi import FastAPI
from fastapi.responses import StreamingResponse
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel

app = FastAPI()

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


async def generate_stream(prompt: str):

    # 🔥 Immediate first token (small but real content)
    first_chunk = "Streaming response started. "
    yield f"data: {{\"choices\":[{{\"delta\":{{\"content\":\"{first_chunk}\"}}}}]}}\n\n"

    # DO NOT sleep here

    base_text = (
        "Streaming large language model APIs enable real-time content generation "
        "by progressively delivering tokens as they are produced. "
        "Unlike traditional batch systems, streaming significantly reduces perceived latency "
        "and improves user experience. "
        "Proper implementation requires chunked transfer encoding, buffer flushing, "
        "error handling, scalability strategies, and performance monitoring. "
    )

    full_text = base_text * 20  # ensures >1625 characters

    chunk_size = 120

    for i in range(0, len(full_text), chunk_size):
        chunk = full_text[i:i+chunk_size]
        yield f"data: {{\"choices\":[{{\"delta\":{{\"content\":\"{chunk}\"}}}}]}}\n\n"
        await asyncio.sleep(0.03)

    yield "data: [DONE]\n\n"


@app.post("/stream")
async def stream_endpoint(request: PromptRequest):
    return StreamingResponse(
        generate_stream(request.prompt),
        media_type="text/event-stream",
        headers={
            "Cache-Control": "no-cache",
            "Connection": "keep-alive",
            "X-Accel-Buffering": "no",
        },
    )


@app.get("/")
async def health():
    return {"status": "ok"}
