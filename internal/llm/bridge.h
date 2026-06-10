#ifndef NEXO_LLM_BRIDGE_H
#define NEXO_LLM_BRIDGE_H
#include <stdint.h>

typedef struct NexoLLM NexoLLM;

typedef struct {
    char *text;
    int n_tokens;
    int64_t timestamp_ms;
} NexoResponse;

NexoLLM* nexo_llm_init(const char *model_path, int n_ctx, int n_threads);
NexoResponse* nexo_llm_generate(NexoLLM *llm, const char *prompt, int max_tokens, float temperature);
void nexo_llm_free_response(NexoResponse *res);
void nexo_llm_free(NexoLLM *llm);

#endif
