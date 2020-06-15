package 反射;


import java.lang.reflect.Constructor;
import java.lang.reflect.Field;
import java.lang.reflect.Method;

public class ReflectClass {
    public static void main(String[] args) {
        ReflectClass.reflectPrivateMethod();
        ReflectClass.reflectPrivateConstructor();
        ReflectClass.reflectPrivateField();
        ReflectClass.reflectPrivateMethod();
    }


    // 创建对象
    static void reflectNewInstance() {
        try {
            Class<?> classBook = Class.forName("F:\\面试突击\\test\\src\\反射\\Book.java");
            Object objectBook = classBook.newInstance();
            Book book = (Book) objectBook;
            book.setName("Android进阶之光");
            book.setAuthor("刘望舒");
            System.out.println(book.toString());
        } catch (Exception ex) {
            ex.printStackTrace();
        }
    }

    // 反射私有的构造方法
    public static void reflectPrivateConstructor() {
        try {
            Class<?> classBook = Class.forName("F:\\面试突击\\test\\src\\反射\\Book.java");
            Constructor<?> declaredConstructorBook = classBook.getDeclaredConstructor(String.class,String.class);
            declaredConstructorBook.setAccessible(true);
            Object objectBook = declaredConstructorBook.newInstance("Android开发艺术探索","任玉刚");
            Book book = (Book) objectBook;
            System.out.println(book.toString());
        } catch (Exception ex) {
            ex.printStackTrace();
        }
    }

    // 反射私有属性
    public static void reflectPrivateField() {
        try {
            Class<?> classBook = Class.forName("F:\\面试突击\\test\\src\\反射\\Book.java");
            Object objectBook = classBook.newInstance();
            Field fieldTag = classBook.getDeclaredField("TAG");
            fieldTag.setAccessible(true);
            String tag = (String) fieldTag.get(objectBook);
            System.out.println(tag);
        } catch (Exception ex) {
            ex.printStackTrace();
        }
    }

    // 反射私有方法
    public static void reflectPrivateMethod() {
        try {
            Class<?> classBook = Class.forName("F:\\面试突击\\test\\src\\反射\\Book.java");
            Method methodBook = classBook.getDeclaredMethod("declaredMethod",int.class);
            methodBook.setAccessible(true);
            Object objectBook = classBook.newInstance();
            String string = (String) methodBook.invoke(objectBook,0);
            System.out.println(string);
        } catch (Exception ex) {
            ex.printStackTrace();
        }
    }
}