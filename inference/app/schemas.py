from pydantic import BaseModel, Field
from typing import List, Dict, Any


class EmbeddingResponse(BaseModel):
    face_embedding: List[float] = Field(..., min_length=512, max_length=512)
    name_embedding: List[float] = Field(..., min_length=768, max_length=768)
    face_dim: int = 512
    name_dim: int = 768
    model_info: str
    liveness: Dict[str, Any]
