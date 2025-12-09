import os
import torch
from datasets import load_dataset
from transformers import (
    AutoModelForSeq2SeqLM,
    AutoTokenizer,
    Seq2SeqTrainingArguments,
    Seq2SeqTrainer,
    DataCollatorForSeq2Seq,
)

# --- 1. CONFIGURATION & CONSTANTS ---
TRAIN_CSV_PATH = "/kaggle/input/dataset/samsum-train.csv"
VALIDATION_CSV_PATH = "/kaggle/input/dataset/samsum-validation.csv"
TEST_CSV_PATH = "/kaggle/input/dataset/samsum-test.csv"

MODEL_CHECKPOINT = "t5-base" 
# Use Seq2Seq Architecture -> Input text → Encoder → Compressed meaning → Decoder → Output text

OUTPUT_DIR = "content-summarizer-final"
PREFIX = "summarize: "
MAX_INPUT_LENGTH = 1024
MAX_TARGET_LENGTH = 128
BATCH_SIZE = 2
NUM_EPOCHS = 3
LEARNING_RATE = 2e-5 # You could say step size, if too high, might skip too much


# --- 2. LOAD TOKENIZER (Global) ---
tokenizer = AutoTokenizer.from_pretrained(MODEL_CHECKPOINT)
# Tokenizer used for encoding n decoding text


# --- 3. PREPROCESSING FUNCTION ---
def preprocess_function(examples):
    """
    Takes a batch of 'dialogue' and 'summary' examples,
    and tokenizes them for T5.
    """
    # Check for None values and replace with empty string
    dialogues = [doc if doc is not None else "" for doc in examples["dialogue"]]
    summaries = [doc if doc is not None else "" for doc in examples["summary"]]

    # the columns are 'dialogue' and 'summary'
    inputs = [PREFIX + doc for doc in dialogues]
    targets = summaries # Use the cleaned summaries

    # Tokenize inputs
    model_inputs = tokenizer(
        inputs,
        max_length=MAX_INPUT_LENGTH,
        truncation=True,
        padding="max_length" # Pad so all sequences is fixed length
    )

    # Tokenize summaries (labels)
    labels = tokenizer(
        text_target=targets,
        max_length=MAX_TARGET_LENGTH,
        truncation=True,
        padding="max_length"
    )

    # Add the tokenized labels to our model inputs
    model_inputs["labels"] = labels["input_ids"]
    return model_inputs



# --- 4. MAIN TRAINING & USAGE FUNCTION ---
def main():
    print(f"--- Starting: Loading Model {MODEL_CHECKPOINT} ---")
    
    model = AutoModelForSeq2SeqLM.from_pretrained(MODEL_CHECKPOINT)

    print(f"--- Loading Datasets ---")
    
    data_files = {
        "train": TRAIN_CSV_PATH,
        "validation": VALIDATION_CSV_PATH,
        "test": TEST_CSV_PATH
    }
    
    try:
        raw_datasets = load_dataset("csv", data_files=data_files)
    except FileNotFoundError:
        print(f"ERROR: Could not find dataset files.")
        print("Please make sure your CSV files are in the same directory and named:")
        print(f"Train: {TRAIN_CSV_PATH}")
        print(f"Validation: {VALIDATION_CSV_PATH}")
        print(f"Test: {TEST_CSV_PATH}")
        return

    print(f"Training examples: {len(raw_datasets['train'])}")
    print(f"Validation examples: {len(raw_datasets['validation'])}")
    print(f"Test examples: {len(raw_datasets['test'])}")


    print("--- Tokenizing and Preprocessing Data ---")
    # Apply the preprocessing function to the entire dataset dict (train, valid, test)
    tokenized_datasets = raw_datasets.map(preprocess_function, batched=True)

    # --- 5. SET UP THE TRAINER ---
    
    training_args = Seq2SeqTrainingArguments(
        output_dir=OUTPUT_DIR,
        eval_strategy="epoch", # Eval every epoch
        learning_rate=LEARNING_RATE,
        per_device_train_batch_size=BATCH_SIZE,
        per_device_eval_batch_size=BATCH_SIZE,
        weight_decay=0.01, # L2 Regularization (Weight Decay) -> To prevent model from overfitting by penalizing large weights in the model
        save_total_limit=3, # Keep last 3 model checkpoints 
        num_train_epochs=NUM_EPOCHS,
        predict_with_generate=True,
        fp16=torch.cuda.is_available(), # Enable FP16 if GPU is there
        push_to_hub=False, # No pushing to huggingface

        report_to="none",   # Disable wandb/tensorboard logs
        logging_steps=50,

        gradient_accumulation_steps=4 # Accumulate gradients to simulate larger batch size 
    )

    # Create dynamic padding & Formatting 
    data_collator = DataCollatorForSeq2Seq(tokenizer, model=model)

    # Handles training loop, evaluation loop, gradient steps..
    trainer = Seq2SeqTrainer(
        model=model,
        args=training_args,
        train_dataset=tokenized_datasets["train"],
        eval_dataset=tokenized_datasets["validation"],
        tokenizer=tokenizer,
        data_collator=data_collator,
    )

    # --- 6. START FINE-TUNING ---
    print("\n--- STARTING MODEL FINE-TUNING ---")
    trainer.train()
    print("--- FINE-TUNING COMPLETE ---")

    # --- 7. EVALUATE ON TEST SET ---
    print("\n--- EVALUATING ON TEST DATASET ---")
    test_results = trainer.evaluate(eval_dataset=tokenized_datasets["test"])
    print("--- Test Results ---")
    print(test_results)

    # --- 8. SAVE THE FINAL MODEL ---
    print(f"--- Saving model and tokenizer to {OUTPUT_DIR} ---")
    trainer.save_model(OUTPUT_DIR)
    tokenizer.save_pretrained(OUTPUT_DIR)
    print("--- Model saved successfully! ---")


    # --- 9. TEST THE SAVED MODEL (INFERENCE) ---
    print(f"\n--- Loading fine-tuned model from {OUTPUT_DIR} for a test ---")
    
    saved_tokenizer = AutoTokenizer.from_pretrained(OUTPUT_DIR)
    saved_model = AutoModelForSeq2SeqLM.from_pretrained(OUTPUT_DIR)

    test_caption = """
    Hannah: Hey, do you have Betty's number?
    Amanda: Lemme check
    Hannah: <file_gif>
    Amanda: Sorry, can't...
    """

    print(f"\nOriginal Caption:\n{test_caption}")
    
    inputs = saved_tokenizer(
        PREFIX + test_caption,
        return_tensors="pt",
        max_length=MAX_INPUT_LENGTH,
        truncation=True
    )

    summary_ids = saved_model.generate(
        inputs["input_ids"],
        num_beams=4,
        max_length=150,
        min_length=10,
        early_stopping=True
    )

    generated_summary = saved_tokenizer.decode(
        summary_ids[0],
        skip_special_tokens=True
    )

    print(f"\nGenerated Summary:\n{generated_summary}")


if __name__ == "__main__":
    main()