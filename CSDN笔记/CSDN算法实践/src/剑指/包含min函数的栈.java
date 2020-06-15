package 剑指;

import java.util.Stack;

public class 包含min函数的栈 {
    Stack<Integer> stack = new Stack<Integer>();
    Stack<Integer> min_stack = new Stack<Integer>(); // record min

    public void push(int node) {
        stack.push(node);
        if (min_stack.empty() || node < min_stack.peek()){
            min_stack.add(node);
        }
    }

    public void pop() {
        Integer pop = stack.pop();
        if (pop == min_stack.peek()){
            min_stack.pop();
        }
    }

    public int top() {
        return stack.pop();
    }

    public int min() {
        return min_stack.peek();
    }
}
