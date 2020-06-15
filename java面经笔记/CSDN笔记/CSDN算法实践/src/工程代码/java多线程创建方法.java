package 工程代码;

import java.util.concurrent.Callable;
import java.util.concurrent.FutureTask;

public class java多线程创建方法 {
    public static void main(String[] args) {

        // 1、MyThread类与Thread类平级，因为都是实现自Runnable，重写了run抽象方法，传递的是一个Runnable对象
        Thread thread = new Thread(new MyThread());
        thread.start();

        // 2、A继承了Thread类，重写了Thread列的run方法， Thread本身就有一个target Runnable对象
        new A().start();

        // 3、匿名内部类， 传递的是一个Runnable对象
        new Thread(new Runnable() {
            @Override
            public void run() {
                System.out.println("匿名内部类的方式创建线程");
            }
        }).start();

        // 4、FutureTask继承自RunnableFuture，而后者继承自Runnable
        FutureTask<Integer> futureTask = new FutureTask<>(new C());
        // 传递的仍是一个Runnable对象
        Thread thread2 = new Thread(futureTask);
        thread2.start();


        //获取线程数
        ThreadGroup threadGroup = Thread.currentThread().getThreadGroup();
        while(threadGroup.getParent() != null){
            threadGroup = threadGroup.getParent();
        }
        int totalThread = threadGroup.activeCount();
        System.out.println("当前线程数："+totalThread);

    }
}

class MyThread implements Runnable{
    @Override
    public void run(){
        System.out.println("实现Runnbale的方式……");
    }
}

class A extends Thread {
    @Override
    public void run() {
        System.out.println("继承Thread类的线程……");
    }
}

class C implements Callable<Integer> {
    @Override
    public Integer call() throws Exception {
        System.out.println("实现callable的形式创建的线程");
        return 1024;
    }
}
