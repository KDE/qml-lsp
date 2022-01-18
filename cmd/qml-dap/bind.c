extern void goCallback(void*, char*);

void callbackGo(void* userData, char* data)
{
	goCallback(userData, data);
}