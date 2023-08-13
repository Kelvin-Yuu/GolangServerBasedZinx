package zasync_op

/*
	<异步IO模块>

1. 业务线程执行业务操作，发送一个IO请求，由IO线程来完成写库，如果写完库之后，还有其他操作？
	a. 接下来的逻辑就在IO线程里执行了；
	b. 回到不是原来的业务线程，而是另一个业务线程里执行；
	这两种情况，相当于一部分业务逻辑在A线程里，一部分业务逻辑在B线程里；两个线程同时操作一块内存区域，会出现脏读写问题！

2. 因此，必须回到所属的业务线程里执行，【业务逻辑原先是由谁来执行的，那么 IO 操作完成之后，继续交还给原来的人去执行。】

3. 使用方法：
	a. 调用 Process 选择一个异步worker进行异步IO操作逻辑；
	b. 在异步IO逻辑中设置需要共享的变量，及异步返回的结果：asyncResult.SetReturnedObj
	c. 注册设置异步回调，即回到原本的业务线程里继续进行后续的操作：asyncResult.OnComplete

*/
