from app.name_embedder import name_embedding_768


def test_name_embedding_dim_and_determinism():
    a = name_embedding_768("John Doe")
    b = name_embedding_768(" john   doe ")
    assert len(a) == 768
    assert len(b) == 768
    assert a == b
    assert abs(sum(x * x for x in a) - 1.0) < 1e-6
