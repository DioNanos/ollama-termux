// Micro test: load Android system Vulkan loader directly and list GPUs.
// Goal: bypass Termux linker namespace restriction on /vendor/lib64/hw/.

#include <vulkan/vulkan.h>
#include <stdio.h>
#include <string.h>

int main(void) {
    VkApplicationInfo app = {
        .sType = VK_STRUCTURE_TYPE_APPLICATION_INFO,
        .pApplicationName = "vulkan_probe",
        .apiVersion = VK_API_VERSION_1_1,
    };
    VkInstanceCreateInfo info = {
        .sType = VK_STRUCTURE_TYPE_INSTANCE_CREATE_INFO,
        .pApplicationInfo = &app,
    };
    VkInstance inst = VK_NULL_HANDLE;
    VkResult r = vkCreateInstance(&info, NULL, &inst);
    if (r != VK_SUCCESS) {
        printf("vkCreateInstance failed: VkResult=%d\n", r);
        return 1;
    }

    uint32_t n = 0;
    vkEnumeratePhysicalDevices(inst, &n, NULL);
    printf("physical devices: %u\n", n);
    if (n == 0) {
        vkDestroyInstance(inst, NULL);
        return 2;
    }

    if (n > 8) n = 8;
    VkPhysicalDevice devs[8];
    vkEnumeratePhysicalDevices(inst, &n, devs);

    const char *types[] = {"OTHER", "INTEGRATED_GPU", "DISCRETE_GPU", "VIRTUAL_GPU", "CPU"};
    for (uint32_t i = 0; i < n; i++) {
        VkPhysicalDeviceProperties p;
        vkGetPhysicalDeviceProperties(devs[i], &p);
        const char *t = (p.deviceType <= 4) ? types[p.deviceType] : "UNKNOWN";
        printf("  [%u] name=%s\n", i, p.deviceName);
        printf("       type=%s  api=%u.%u.%u  driver=0x%x  vendor=0x%x device=0x%x\n",
            t,
            VK_VERSION_MAJOR(p.apiVersion), VK_VERSION_MINOR(p.apiVersion), VK_VERSION_PATCH(p.apiVersion),
            p.driverVersion, p.vendorID, p.deviceID);
    }

    vkDestroyInstance(inst, NULL);
    return 0;
}
