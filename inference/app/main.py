from fastapi import FastAPI, File, Form, HTTPException, UploadFile

from app.face_embedder import ArcFaceEmbedder, FaceEmbeddingError
from app.name_embedder import name_embedding_768
from app.schemas import EmbeddingResponse

app = FastAPI(title="Visual KYC Inference Service", version="0.1.0")
face_embedder = ArcFaceEmbedder()


@app.get("/health")
def health():
    return {"status": "ok", "mode": face_embedder.mode}


@app.post("/embed", response_model=EmbeddingResponse)
async def embed(image: UploadFile = File(...), name: str = Form(...)):
    image_bytes = await image.read()
    if not image_bytes:
        raise HTTPException(status_code=400, detail="image is required")
    if not name.strip():
        raise HTTPException(status_code=400, detail="name is required")
    try:
        face_vec, model_info, liveness = face_embedder.embed(image_bytes)
    except FaceEmbeddingError as exc:
        raise HTTPException(status_code=422, detail=str(exc)) from exc

    name_vec = name_embedding_768(name)
    return EmbeddingResponse(
        face_embedding=face_vec,
        name_embedding=name_vec,
        face_dim=len(face_vec),
        name_dim=len(name_vec),
        model_info=model_info,
        liveness=liveness,
    )
