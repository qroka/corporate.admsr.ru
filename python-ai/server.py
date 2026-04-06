"""
ADMSR Corporate AI Chat Server
================================
Запускает FastAPI сервер на порту 8000 и использует Ollama
для инференса модели Qwen2.5:7b (instruction-tuned).

Схема взаимодействия:
  Браузер → Vue SPA → /api/chat.php → http://127.0.0.1:8000/chat → Ollama → Qwen2.5:7b
"""

import logging
from contextlib import asynccontextmanager
from typing import Literal

import ollama
import uvicorn
from fastapi import FastAPI, HTTPException, Request
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel, field_validator

# ─── Настройки ────────────────────────────────────────────────────────────────

MODEL_NAME = "qwen2.5:7b"       # ollama pull qwen2.5:7b
HOST       = "0.0.0.0"
PORT       = 8000

SYSTEM_PROMPT = """Ты корпоративный AI-ассистент компании ADMSR (Администрация).
Твоя задача — помогать сотрудникам с рабочими вопросами: кадровые процедуры,
корпоративные мероприятия, внутренние регламенты, рабочие задачи и прочее.

Правила:
- Отвечай только на русском языке.
- Будь вежливым, лаконичным и по делу.
- Если не знаешь ответа — честно скажи об этом.
- Не давай медицинских, юридических или финансовых советов.
- Максимум один-два абзаца, если вопрос не требует подробного ответа."""

# ─── Логирование ──────────────────────────────────────────────────────────────

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s  %(levelname)-8s  %(message)s",
    datefmt="%H:%M:%S",
)
log = logging.getLogger(__name__)

# ─── Lifespan: проверяем Ollama при старте ────────────────────────────────────

@asynccontextmanager
async def lifespan(app: FastAPI):
    log.info("Проверяю доступность Ollama и модели '%s'...", MODEL_NAME)
    try:
        models = ollama.list()
        names = [m.model for m in models.models]
        if not any(MODEL_NAME in n for n in names):
            log.warning("Модель '%s' не найдена. Запускаю скачивание...", MODEL_NAME)
            log.warning("Это может занять несколько минут (~4 ГБ).")
            ollama.pull(MODEL_NAME)
            log.info("Модель '%s' успешно загружена.", MODEL_NAME)
        else:
            log.info("Модель '%s' найдена. Сервер готов.", MODEL_NAME)
    except Exception as exc:
        log.error("Не удалось подключиться к Ollama: %s", exc)
        log.error("Убедитесь что Ollama запущена: systemctl start ollama")
    yield
    log.info("Сервер остановлен.")

# ─── Приложение ───────────────────────────────────────────────────────────────

app = FastAPI(
    title="ADMSR AI Chat",
    version="1.0.0",
    description="Корпоративный AI-ассистент на базе Qwen2.5:7b",
    lifespan=lifespan,
)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["http://localhost:5173", "http://localhost:5174", "http://127.0.0.1"],
    allow_methods=["POST", "GET", "OPTIONS"],
    allow_headers=["Content-Type", "Authorization"],
)

# ─── Схемы данных ─────────────────────────────────────────────────────────────

class Message(BaseModel):
    role: Literal["user", "assistant"]
    content: str

    @field_validator("content")
    @classmethod
    def trim_content(cls, v: str) -> str:
        return v.strip()[:4096]

class ChatRequest(BaseModel):
    messages: list[Message]

    @field_validator("messages")
    @classmethod
    def validate_messages(cls, v: list[Message]) -> list[Message]:
        if not v:
            raise ValueError("messages не может быть пустым")
        return v[-20:]  # контекст: последние 20 сообщений

class ChatResponse(BaseModel):
    success: bool
    message: str
    model: str
    prompt_tokens: int | None = None
    response_tokens: int | None = None

# ─── Эндпоинты ────────────────────────────────────────────────────────────────

@app.get("/health")
async def health():
    """Проверка работоспособности сервера."""
    try:
        models = ollama.list()
        names = [m.model for m in models.models]
        model_ready = any(MODEL_NAME in n for n in names)
        return {
            "status": "ok",
            "model": MODEL_NAME,
            "model_ready": model_ready,
            "available_models": names,
        }
    except Exception as exc:
        return {"status": "error", "detail": str(exc)}


@app.post("/chat", response_model=ChatResponse)
async def chat(request: ChatRequest, req: Request):
    """Отправить сообщения и получить ответ от Qwen."""
    client_ip = req.client.host if req.client else "unknown"
    log.info("[%s] Запрос: %d сообщений", client_ip, len(request.messages))

    # Собираем историю с системным промптом
    history = [{"role": "system", "content": SYSTEM_PROMPT}]
    for msg in request.messages:
        history.append({"role": msg.role, "content": msg.content})

    try:
        response = ollama.chat(
            model=MODEL_NAME,
            messages=history,
            options={
                "temperature": 0.7,
                "num_predict": 1024,
                "top_p": 0.9,
            },
        )

        reply = response.message.content.strip()
        log.info("[%s] Ответ: %d символов", client_ip, len(reply))

        return ChatResponse(
            success=True,
            message=reply,
            model=response.model,
            prompt_tokens=response.prompt_eval_count,
            response_tokens=response.eval_count,
        )

    except ollama.ResponseError as exc:
        log.error("Ошибка Ollama: %s", exc)
        raise HTTPException(status_code=503, detail=f"Ошибка модели: {exc.error}")
    except Exception as exc:
        log.error("Непредвиденная ошибка: %s", exc)
        raise HTTPException(status_code=500, detail="Внутренняя ошибка сервера")


# ─── Запуск напрямую ──────────────────────────────────────────────────────────

if __name__ == "__main__":
    uvicorn.run(
        "server:app",
        host=HOST,
        port=PORT,
        reload=False,
        log_level="info",
    )
