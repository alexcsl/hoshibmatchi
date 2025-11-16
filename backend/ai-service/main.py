import uvicorn
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
import os
import torch # Import torch
from transformers import T5Tokenizer, T5ForConditionalGeneration # Use T5 specific imports

app = FastAPI()

# --- Configuration for your trained model ---
# Change this to your Hugging Face model ID
MODEL_NAME = os.getenv("MODEL_NAME", "alexcsl10/baption")
MODEL_PATH = os.getenv("MODEL_PATH", None)  # Set via environment variable if using local model

tokenizer = None
model = None

# --- Pydantic Models ---
class CaptionRequest(BaseModel):
    caption: str

class SummaryResponse(BaseModel):
    summary: str

# --- Startup Event: Load the model when FastAPI starts ---
@app.on_event("startup")
async def load_model():
    global tokenizer, model
    try:
        # Determine the model source (local path or Hugging Face)
        model_source = MODEL_PATH if MODEL_PATH else MODEL_NAME
        print(f"Attempting to load tokenizer from {model_source}")
        
        # Use T5Tokenizer specifically with legacy=True to avoid tokenizer.json issues
        from transformers import T5Tokenizer
        tokenizer = T5Tokenizer.from_pretrained(model_source, legacy=True)
        
        print(f"Attempting to load model from {model_source}")
        # Use T5ForConditionalGeneration specifically
        from transformers import T5ForConditionalGeneration
        model = T5ForConditionalGeneration.from_pretrained(model_source)
        
        # Move model to GPU if available
        if torch.cuda.is_available():
            model.to("cuda")
            print("Model moved to GPU (CUDA)")
        else:
            print("Model loaded on CPU")
        model.eval() # Set model to evaluation mode
        print(f"AI Model and Tokenizer loaded successfully from {model_source}!")
    except Exception as e:
        print(f"Failed to load AI model or tokenizer: {e}")
        import traceback
        traceback.print_exc()
        # Depending on criticality, you might want to raise an exception here
        # or have /summarize endpoint handle the missing model.
        tokenizer = None
        model = None

@app.post("/summarize", response_model=SummaryResponse)
async def summarize_caption(request: CaptionRequest):
    """
    Summarizes a given post caption using the loaded AI model.
    """
    if tokenizer is None or model is None:
        raise HTTPException(status_code=503, detail="AI model not loaded. Service unavailable.")

    caption = request.caption

    # --- Step 2: Generate summary using the model ---
    with torch.no_grad(): # Disable gradient calculation for inference
        # Add "summarize: " prefix for T5 models
        input_text = f"summarize: {caption}"
        inputs = tokenizer(input_text, return_tensors="pt", max_length=1024, truncation=True)
        
        # Move inputs to GPU if model is on GPU
        if torch.cuda.is_available():
            inputs = {k: v.to("cuda") for k, v in inputs.items()}
        
        # Generation parameters can be tuned for better results
        summary_ids = model.generate(
            inputs.input_ids, 
            num_beams=4, 
            max_length=150, # Max length of generated summary
            min_length=30,  # Min length of generated summary
            early_stopping=True 
        )

    # --- Step 3: Decode the generated summary ---
    summary_text = tokenizer.decode(summary_ids[0], skip_special_tokens=True)
    
    return SummaryResponse(summary=summary_text)

@app.get("/health")
async def health_check():
    if tokenizer and model:
        return {"status": "ok", "model_loaded": True}
    else:
        raise HTTPException(status_code=503, detail="AI model not loaded")

if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=9008)