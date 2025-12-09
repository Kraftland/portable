#include <stdio.h>
#include

int main() {
  printf("Welcome to Portable configuration generator.\n");
  printf("Please read the documentation first before packaging.\n");

  int appIDAccepted = 0;
  char appID[] = "d";
  int dotPresence = 0;
  while (appIDAccepted == 0) {
    printf("Enter the desired application identifier in reverse DNS order: ");
    scanf("%s", appID);
    printf("Got user input: %s\n", appID);
    int appIDLen = sizeof(appID) / sizeof(appID[0]);
    printf("Length of app ID: %i\n", appIDLen);
    unsigned long long int i;
    for (i = 0; i < appIDLen; i++) {
      char loopChar = appID[i];
      printf("current character: %c\n", loopChar);
      if (loopChar == '.') {
	dotPresence++;
	printf("Detected dot presence\n");
      }
    }
    if (dotPresence > 2) {
      break;
    }
  }
  printf("Calculated dotPresence: %i", dotPresence);
}
