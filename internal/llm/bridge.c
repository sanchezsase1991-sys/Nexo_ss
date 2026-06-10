#include "bridge.h"
#include "../../llama.cpp/include/llama.h"
#include <stdlib.h>
#include <string.h>

struct NexoLLM {
    struct llama_model *model;
    struct llama_context *ctx;
    const struct llama_vocab *vocab;
    int n_ctx;
};

NexoLLM* nexo_llm_init(const char *model_path, int n_ctx, int n_threads) {
    NexoLLM *llm = calloc(1, sizeof(NexoLLM));
    if (!llm) return NULL;

    struct llama_model_params mparams = llama_model_default_params();
    llm->model = llama_load_model_from_file(model_path, mparams);
    if (!llm->model) { free(llm); return NULL; }

    struct llama_context_params cparams = llama_context_default_params();
    cparams.n_ctx = n_ctx;
    cparams.n_threads = n_threads;
    cparams.n_batch = 512;
    llm->ctx = llama_new_context_with_model(llm->model, cparams);
    if (!llm->ctx) { llama_free_model(llm->model); free(llm); return NULL; }

    llm->vocab = llama_model_get_vocab(llm->model);
    llm->n_ctx = n_ctx;
    return llm;
}

NexoResponse* nexo_llm_generate(NexoLLM *llm, const char *prompt, int max_tokens, float temperature) {
    if (!llm || !prompt) return NULL;

    int prompt_len = strlen(prompt);
    int n_tokens = llama_tokenize(llm->vocab, prompt, prompt_len, NULL, 0, true, true);
    if (n_tokens < 0) return NULL;

    int *tokens = malloc(n_tokens * sizeof(int));
    llama_tokenize(llm->vocab, prompt, prompt_len, tokens, n_tokens, true, true);

    char *output = malloc(max_tokens * 8);
    memset(output, 0, max_tokens * 8);
    int output_pos = 0;

    int n_processed = 0;
    while (n_processed < n_tokens) {
        int batch_size = (n_tokens - n_processed < 512) ? (n_tokens - n_processed) : 512;
        llama_batch batch = llama_batch_get_one(tokens + n_processed, batch_size);
        if (llama_decode(llm->ctx, batch) != 0) break;
        n_processed += batch_size;
    }

    int prev_token = tokens[n_tokens - 1];
    int total_tokens = 0;
    for (int i = 0; i < max_tokens; i++) {
        int next_token = llama_sample_token(llm->ctx, prev_token);
        if (next_token == llama_token_eos(llm->model)) break;

        char piece[32];
        int piece_len = llama_token_to_piece(llm->vocab, next_token, piece, sizeof(piece), 0, true);
        if (piece_len < 0) break;

        if (output_pos + piece_len < max_tokens * 8) {
            memcpy(output + output_pos, piece, piece_len);
            output_pos += piece_len;
        }

        total_tokens++;
        prev_token = next_token;

        llama_batch single = llama_batch_get_one(&next_token, 1);
        if (llama_decode(llm->ctx, single) != 0) break;
    }

    free(tokens);

    NexoResponse *res = malloc(sizeof(NexoResponse));
    res->text = output;
    res->n_tokens = total_tokens;
    return res;
}

void nexo_llm_free_response(NexoResponse *res) {
    if (res) { free(res->text); free(res); }
}

void nexo_llm_free(NexoLLM *llm) {
    if (llm) {
        if (llm->ctx) llama_free(llm->ctx);
        if (llm->model) llama_free_model(llm->model);
        free(llm);
    }
}
