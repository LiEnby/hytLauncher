#include <stdint.h>
#include <stdio.h>

#include "shared.h"
#include "cs_string.h"

static int num_swaps = 0;

typedef struct swapEntry {
    csString new;
    csString old;
} swapEntry;


void overwrite(csString* old, csString* new) {    
    int prev = get_prot(old);
    printf("Previous protection level: %x\n", prev);
    
    if (change_prot((uintptr_t)old, get_rw_perms()) == 0) {
        printf("copying .. %p to .. %p\n", new, old);
        
        int sz = get_size_ptr(new);
        memcpy(old, new, sz);
#ifdef _WIN32
        //memset(&((uint8_t*)old)[sz], 0, 2);
#endif

    }

    change_prot((uintptr_t)old, prev);
}


void allowOfflineInOnline(uint8_t* mem) {

    if (PATTERN_PLATFORM) {

        printf("Found debug check @ %p\n", mem);

#ifdef __linux__
        void* target = &mem[13];
#elif _WIN32
        void* target = &mem[15];
#endif
        int prev = get_prot(target);
        printf("Previous protection level %x\n", prev);

       
        if (change_prot((uintptr_t)target, get_rw_perms()) == 0) {
            memset(target, 0x90, 0x6);
        }
        else {
            printf("Failed to change protection level\n");
        }
        
        if (change_prot((uintptr_t)target, prev) != 0) {
            printf("Failed to change protection level back\n");
        }

    }

}


void swap(uint8_t* mem, csString* old, csString* new) {
    if (memcmp(mem, old, get_size_ptr(old)) == 0) {
        overwrite((csString*)mem, new);
        num_swaps++;
    }
}


void changeServers() {

    swapEntry swaps[] = {
        {.old = make_csstr(L"https://account-data."), .new = make_csstr(L"http://127.0.0")},
        {.old = make_csstr(L"https://sessions."),     .new = make_csstr(L"http://127.0.0")},
        {.old = make_csstr(L"https://telemetry."),    .new = make_csstr(L"http://127.0.0")},
        {.old = make_csstr(L"https://tools."),        .new = make_csstr(L"http://127.0.0")},
        {.old = make_csstr(L"hytale.com"),            .new = make_csstr(L".1:59313")},
        {.old = make_csstr(L"authenticated"),         .new = make_csstr(L"offline")},
    };

    int totalSwaps = (sizeof(swaps) / sizeof(swapEntry));
    printf("Total Swaps: %x\n", totalSwaps);
   
    modinfo inf = get_base();
    uint8_t* memory = inf.start;

    for (size_t i = 0; i < inf.sz; i++) {
        // allow online mode in offline mode.
        allowOfflineInOnline(&memory[i]);
        
        for (int sw = 0; sw < totalSwaps; sw++) {
            swap(&memory[i], &swaps[sw].old, &swaps[sw].new);
        }

        if (num_swaps >= totalSwaps) break;
    }


    printf("TOTAL SWAPS: %d\n", num_swaps);
}
